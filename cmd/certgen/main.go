package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	certName = "cert.pem"
	keyName  = "key.pem"
)

func generateCert(outDir string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate rsa key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"Self Signed"},
		},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	certPath := filepath.Join(outDir, certName)
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	keyPath := filepath.Join(outDir, keyName)
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

func main() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDir := filepath.Dir(execPath)

	var outDir string
	flag.StringVar(&outDir, "out-dir", execDir, "")
	flag.Parse()

	if err := generateCert(outDir); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Certificate generated: %s\n", filepath.Join(outDir, certName))
	fmt.Printf("Key generated: %s\n", filepath.Join(outDir, keyName))
}
