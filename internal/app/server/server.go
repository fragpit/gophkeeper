package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	"github.com/fragpit/gophkeeper/internal/api/router"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/fragpit/gophkeeper/internal/service/cryptor"
	"github.com/fragpit/gophkeeper/internal/service/healthcheck"
	"github.com/fragpit/gophkeeper/internal/service/items"
	"github.com/fragpit/gophkeeper/internal/storage/postgresql"
	"github.com/fragpit/gophkeeper/internal/storage/s3"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg *config.ServerConfig) error {
	slog.Info("staring gophkeeper server", "address", cfg.Address)

	pgStorage, err := postgresql.NewStorage(ctx, cfg.DatabaseURI)
	if err != nil {
		slog.Error("initialize rdb storage", slog.Any("error", err))
		os.Exit(1)
	}

	objectStorage, err := s3.NewS3Storage(
		ctx,
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
	)
	if err != nil {
		slog.Error("initialize object storage", slog.Any("error", err))
		os.Exit(1)
	}

	routerDeps := buildRouterDeps(cfg, pgStorage, objectStorage)
	router, err := router.NewRouter(routerDeps, cfg)
	if err != nil {
		return fmt.Errorf("init router: %w", err)
	}

	eg := &errgroup.Group{}
	eg.Go(func() error {
		if err := router.Run(ctx); err != nil {
			return fmt.Errorf("run router: %w", err)
		}
		return nil
	})

	err = eg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("server failed", "error", err)
		return fmt.Errorf("run server: %w", err)
	}

	slog.Info("server shutdown gracefully")
	return nil
}

func buildRouterDeps(
	cfg *config.ServerConfig,
	repos *postgresql.Repositories,
	objectStorage *s3.S3Storage,
) router.ServiceDeps {
	cryptoSvc := cryptor.NewCryptor(cfg.MasterKey)

	healthSvc := healthcheck.NewHealthcheckService(repos.Healthcheck)
	authSvc := auth.NewAuthService(
		repos.Users,
		cryptoSvc,
		cfg.JWTSecret,
		cfg.JWTTTL,
	)
	itemsSvc := items.NewItemsService(repos.Items)
	createSvc := items.NewCreateItemService(repos.Items, repos.Users, cryptoSvc)
	createFileSvc := items.NewCreateFileService(
		repos.Items,
		repos.Users,
		cryptoSvc,
		objectStorage,
	)
	getSvc := items.NewGetItemService(repos.Items, repos.Users, cryptoSvc)
	getFileSvc := items.NewGetFileService(
		repos.Items,
		repos.Users,
		cryptoSvc,
		objectStorage,
	)

	return router.ServiceDeps{
		HealthService:     healthSvc,
		AuthService:       authSvc,
		ItemsService:      itemsSvc,
		CreateService:     createSvc,
		CreateFileService: createFileSvc,
		GetService:        getSvc,
		GetFileService:    getFileSvc,
	}
}
