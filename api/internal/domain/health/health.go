package health

import "context"

// Status is the domain view of system health.
type Status struct {
	Database bool
}

// OK reports whether every dependency is healthy.
func (s Status) OK() bool { return s.Database }

// Pinger verifies connectivity to an external dependency.
// Implemented by the infrastructure layer (e.g. the Postgres pool).
type Pinger interface {
	Ping(ctx context.Context) error
}
