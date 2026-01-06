package healthcheck

import (
	"context"
)

//go:generate mockgen -destination ./mocks/health_repo_gen.go . HealthRepository
// HealthRepository pings underlying storage to verify availability.
type HealthRepository interface {
	Ping(ctx context.Context) error
}

// HealthService performs system health checks.
type HealthService struct {
	repo HealthRepository
}

// NewHealthcheckService constructs a HealthService with provided repository.
func NewHealthcheckService(repo HealthRepository) *HealthService {
	return &HealthService{
		repo: repo,
	}
}

// Check performs a health check using the repository.
func (h *HealthService) Check(ctx context.Context) error {
	return h.repo.Ping(ctx)
}
