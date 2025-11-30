package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	gracefulShutdownTimeout = 10 * time.Second
)

// ServiceDeps aggregates all services required by the HTTP router.
type ServiceDeps struct {
	AuthService       handlers.AuthService
	HealthService     handlers.HealthService
	ItemsService      handlers.ItemsService
	CreateService     handlers.CreateService
	CreateFileService handlers.CreateFileService
	GetService        handlers.GetService
	GetFileService    handlers.GetFileService
	UpdateService     handlers.UpdateService
	DeleteService     handlers.DeleteService
	UsersRepo         UserVerifier
}

// Router wraps the Echo HTTP server and its configuration.
type Router struct {
	router *echo.Echo

	ListenAddress string
	TLSCertFile   string
	TLSKeyFile    string
}

// NewRouter constructs a configured Router instance with the provided dependencies.
func NewRouter(
	deps ServiceDeps,
	cfg *config.ServerConfig,
	promReg *prometheus.Registry,
) (*Router, error) {
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		if err := parseTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			return nil, fmt.Errorf("failed to parse TLS data: %w", err)
		}
	}

	e := echo.NewWithConfig(echo.Config{
		Logger: slog.Default(),
	})

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	metricTotalRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_http_requests",
		Help: "Total http request to server",
	})
	if err := promReg.Register(metricTotalRequests); err != nil {
		return nil, fmt.Errorf("register metric: %w", err)
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			metricTotalRequests.Inc()
			return next(c)
		}
	})

	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "Welcome to GophKeeper server")
	})
	e.GET("/health", handlers.NewHealthHandler(deps.HealthService))
	e.GET("/metrics", func(c *echo.Context) error {
		h := promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

		h.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	api := e.Group("/api")
	api.Use(echojwt.WithConfig(echojwt.Config{
		Skipper:    jwtAuthSkipper,
		SigningKey: []byte(cfg.JWTSecret),
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return &auth.Claims{}
		},
	}))
	api.Use(UserExistenceMiddleware(deps.UsersRepo))
	api.POST("/register", handlers.NewAuthRegisterHandler(deps.AuthService))
	api.POST("/login", handlers.NewAuthLoginHandler(deps.AuthService))
	api.GET("/list/items", handlers.NewListHandler(deps.ItemsService))
	api.GET("/get/item", handlers.NewGetHandler(deps.GetService))
	api.GET("/get/file", handlers.NewGetFileHandler(deps.GetFileService))
	api.POST("/create/item", handlers.NewCreateHandler(deps.CreateService))
	api.POST(
		"/create/file",
		handlers.NewCreateFileHandler(deps.CreateFileService),
	)
	api.PUT("/update/item", handlers.NewUpdateHandler(deps.UpdateService))
	api.DELETE("/delete/item", handlers.NewDeleteHandler(deps.DeleteService))

	return &Router{
		router:        e,
		ListenAddress: cfg.Address,
		TLSCertFile:   cfg.TLSCertFile,
		TLSKeyFile:    cfg.TLSKeyFile,
	}, nil
}

// Run starts the HTTP server and waits for shutdown.
func (r *Router) Run(ctx context.Context) error {
	if r.TLSCertFile == "" || r.TLSKeyFile == "" {
		return fmt.Errorf("tls cert or key not provided")
	}

	sc := echo.StartConfig{
		Address:         r.ListenAddress,
		GracefulTimeout: gracefulShutdownTimeout,
		HideBanner:      true,
		HidePort:        true,
		CertFilesystem:  os.DirFS(filepath.Dir(r.TLSCertFile)),
	}
	if err := sc.StartTLS(ctx, r.router, filepath.Base(r.TLSCertFile), filepath.Base(r.TLSKeyFile)); err != nil {
		return fmt.Errorf("start tls server: %w", err)
	}

	return nil
}

func parseTLSConfig(cert, key string) error {
	if cert == "" || key == "" {
		return errors.New("tls cert and key are required")
	}

	if _, err := tls.LoadX509KeyPair(cert, key); err != nil {
		return fmt.Errorf("load tls key pair: %w", err)
	}
	return nil
}

func jwtAuthSkipper(c *echo.Context) bool {
	return c.Path() == "/api/login" || c.Path() == "/api/register"
}
