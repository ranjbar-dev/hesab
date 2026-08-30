package health

import (
	"context"

	domain "hesab/api/internal/domain/health"
)

// Service is the health-check use case.
type Service struct {
	db domain.Pinger
}

// NewService wires the health use case to its dependencies.
func NewService(db domain.Pinger) *Service {
	return &Service{db: db}
}

// Check returns the current health status, probing every dependency.
func (s *Service) Check(ctx context.Context) domain.Status {
	return domain.Status{Database: s.db.Ping(ctx) == nil}
}
