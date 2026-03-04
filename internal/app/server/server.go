package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/fragpit/gophkeeper/cmd/server/config"
	"github.com/fragpit/gophkeeper/internal/api/router"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/fragpit/gophkeeper/internal/service/cryptor"
	"github.com/fragpit/gophkeeper/internal/service/healthcheck"
	"github.com/fragpit/gophkeeper/internal/service/items"
	"github.com/fragpit/gophkeeper/internal/storage/postgresql"
	"github.com/fragpit/gophkeeper/internal/storage/s3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"golang.org/x/sync/errgroup"
)

// Run initializes dependencies and starts the server.
func Run(ctx context.Context, cfg *config.ServerConfig) error {
	slog.Info("starting gophkeeper server", "address", cfg.Address)

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	pgStorage, err := postgresql.NewStorage(ctx, cfg.DatabaseURI)
	if err != nil {
		slog.Error("initialize rdb storage", slog.Any("error", err))
		return fmt.Errorf("init rdb: %w", err)
	}

	objectStorage, err := s3.NewS3Storage(
		ctx,
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3BucketName,
	)
	if err != nil {
		slog.Error("initialize object storage", slog.Any("error", err))
		return fmt.Errorf("init object storage: %w", err)
	}

	routerDeps := buildRouterDeps(cfg, pgStorage, objectStorage)
	router, err := router.NewRouter(routerDeps, cfg, promReg)
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

	eg.Go(func() error {
		<-ctx.Done()
		pgStorage.Close()
		slog.Info("database connection pool closed")
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
		cfg.JWTSecret,
		cfg.JWTTTL,
		cryptoSvc,
		repos.Users,
		repos.Tokens,
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
	updateSvc := items.NewUpdateItemService(repos.Items, repos.Users, cryptoSvc)
	deleteSvc := items.NewDeleteService(repos.Items, objectStorage)

	return router.ServiceDeps{
		HealthService:     healthSvc,
		AuthService:       authSvc,
		ItemsService:      itemsSvc,
		CreateService:     createSvc,
		CreateFileService: createFileSvc,
		GetService:        getSvc,
		GetFileService:    getFileSvc,
		UpdateService:     updateSvc,
		DeleteService:     deleteSvc,
		UsersRepo:         repos.Users,
	}
}
