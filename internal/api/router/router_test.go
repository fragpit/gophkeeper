package router

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	mock_handlers "github.com/fragpit/gophkeeper/internal/api/handlers/mocks"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"
)

func TestNewRouter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deps := createMockDeps(ctrl)
	cfg := &config.ServerConfig{
		Address:   ":8080",
		JWTSecret: "dGVzdC1zZWNyZXQ=",
	}

	reg := prometheus.NewRegistry()

	router, err := NewRouter(deps, cfg, reg)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if router == nil {
		t.Fatal("NewRouter() returned nil router")
	}

	if router.ListenAddress != cfg.Address {
		t.Errorf(
			"Router.ListenAddress = %v, want %v",
			router.ListenAddress,
			cfg.Address,
		)
	}
}

func TestNewRouter_WithTLS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	certFile, keyFile, cleanup := createTempTLSFiles(t)
	defer cleanup()

	deps := createMockDeps(ctrl)
	cfg := &config.ServerConfig{
		Address:     ":8443",
		JWTSecret:   "dGVzdC1zZWNyZXQ=",
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}

	reg := prometheus.NewRegistry()
	router, err := NewRouter(deps, cfg, reg)
	if err != nil {
		t.Fatalf("NewRouter() with TLS error = %v", err)
	}

	if router.TLSCertFile != certFile {
		t.Errorf("Router.TLSCertFile = %v, want %v", router.TLSCertFile, certFile)
	}

	if router.TLSKeyFile != keyFile {
		t.Errorf("Router.TLSKeyFile = %v, want %v", router.TLSKeyFile, keyFile)
	}
}

func TestNewRouter_InvalidTLS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deps := createMockDeps(ctrl)
	cfg := &config.ServerConfig{
		Address:     ":8443",
		JWTSecret:   "dGVzdC1zZWNyZXQ=",
		TLSCertFile: "/nonexistent/cert.pem",
		TLSKeyFile:  "/nonexistent/key.pem",
	}

	reg := prometheus.NewRegistry()
	_, err := NewRouter(deps, cfg, reg)
	if err == nil {
		t.Fatal("NewRouter() with invalid TLS should return error")
	}
}

func TestParseTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		cert    string
		key     string
		wantErr bool
	}{
		{"empty cert and key", "", "", true},
		{"empty cert", "", "key.pem", true},
		{"empty key", "cert.pem", "", true},
		{
			"nonexistent files",
			"/nonexistent/cert.pem",
			"/nonexistent/key.pem",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseTLSConfig(tt.cert, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTLSConfig_ValidFiles(t *testing.T) {
	certFile, keyFile, cleanup := createTempTLSFiles(t)
	defer cleanup()

	err := parseTLSConfig(certFile, keyFile)
	if err != nil {
		t.Errorf("parseTLSConfig() with valid files error = %v", err)
	}
}

func TestJWTAuthSkipper(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"skip login path", "/api/login", true},
		{"skip register path", "/api/register", true},
		{"do not skip protected path", "/api/list/items", false},
		{"do not skip get item path", "/api/get/item", false},
		{"do not skip create item path", "/api/create/item", false},
		{"do not skip root path", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tt.path)

			got := jwtAuthSkipper(c)
			if got != tt.want {
				t.Errorf("jwtAuthSkipper() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createMockDeps(ctrl *gomock.Controller) ServiceDeps {
	return ServiceDeps{
		AuthService:       mock_handlers.NewMockAuthService(ctrl),
		HealthService:     mock_handlers.NewMockHealthService(ctrl),
		ItemsService:      mock_handlers.NewMockItemsService(ctrl),
		CreateService:     mock_handlers.NewMockCreateService(ctrl),
		CreateFileService: mock_handlers.NewMockCreateFileService(ctrl),
		GetService:        mock_handlers.NewMockGetService(ctrl),
		GetFileService:    mock_handlers.NewMockGetFileService(ctrl),
	}
}

func createTempTLSFiles(
	t *testing.T,
) (certFile, keyFile string, cleanup func()) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&priv.PublicKey,
		priv,
	)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certTempFile, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp cert file: %v", err)
	}
	certFile = certTempFile.Name()

	if err := pem.Encode(certTempFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("Failed to encode cert: %v", err)
	}
	_ = certTempFile.Close()

	keyTempFile, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		_ = os.Remove(certFile)
		t.Fatalf("Failed to create temp key file: %v", err)
	}
	keyFile = keyTempFile.Name()

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyTempFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		_ = os.Remove(certFile)
		t.Fatalf("Failed to encode key: %v", err)
	}
	_ = keyTempFile.Close()

	cleanup = func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}

	return certFile, keyFile, cleanup
}
