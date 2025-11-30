package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
)

const (
	gracefulShutdownTimeout = 10 * time.Second
)

type ServiceDeps struct {
	AuthService       handlers.AuthService
	HealthService     handlers.HealthService
	ItemsService      handlers.ItemsService
	CreateService     handlers.CreateService
	CreateFileService handlers.CreateFileService
	GetService        handlers.GetService
	GetFileService    handlers.GetFileService
}

type Router struct {
	router *echo.Echo

	ListenAddress string
	TLSCertFile   string
	TLSKeyFile    string
}

func NewRouter(
	deps ServiceDeps,
	cfg *config.ServerConfig,
) (*Router, error) {
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		if err := parseTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			return nil, fmt.Errorf("failed to parse TLS data: %w", err)
		}
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.StdLogger = slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn)

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to GophKeeper server")
	})
	e.GET("/health", handlers.NewHealthHandler(deps.HealthService))

	api := e.Group("/api")
	api.Use(echojwt.WithConfig(echojwt.Config{
		Skipper:    jwtAuthSkipper,
		SigningKey: []byte(cfg.JWTSecret),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return &auth.Claims{}
		},
	}))
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

	return &Router{
		router:        e,
		ListenAddress: cfg.Address,
		TLSCertFile:   cfg.TLSCertFile,
		TLSKeyFile:    cfg.TLSKeyFile,
	}, nil
}

func (r *Router) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	if r.TLSCertFile == "" && r.TLSKeyFile == "" {
		return fmt.Errorf("tls cert or key not provided")
	}

	eg.Go(func() error {
		if err := r.router.StartTLS(r.ListenAddress, r.TLSCertFile, r.TLSKeyFile); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("start tls server: %w", err)
		}

		return nil
	})

	<-egCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		gracefulShutdownTimeout,
	)
	defer cancel()

	if err := r.router.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("server: %w", err)
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

func jwtAuthSkipper(c echo.Context) bool {
	return c.Path() == "/api/login" || c.Path() == "/api/register"
}
