package repository

import (
	"context"
	"encoding/json"
	"time"

	"windtunnel-release/internal/domain"
)

type ReleaseMatch struct {
	ID        string        `json:"id"`
	Status    domain.Status `json:"status"`
	Owner     string        `json:"owner"`
	UpdatedAt time.Time     `json:"updated_at"`
}
type AuditFilter struct {
	EventType    string
	Role         domain.Role
	Actor        string
	RevisionFrom int64
	RevisionTo   int64
	FromStatus   domain.Status
	ToStatus     domain.Status
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	Limit        int
	Offset       int
}
type StageCount struct {
	Status domain.Status `json:"status"`
	Count  int           `json:"count"`
}
type AuditStats struct {
	Total                int          `json:"total"`
	FailureOrReturnCount int          `json:"failure_or_return_count"`
	DistinctActors       int          `json:"distinct_actors"`
	FirstOccurredAt      *time.Time   `json:"first_occurred_at,omitempty"`
	LastOccurredAt       *time.Time   `json:"last_occurred_at,omitempty"`
	Stages               []StageCount `json:"stages"`
}
type AuditView struct {
	domain.Page[domain.AuditEvent]
	Stats AuditStats `json:"stats"`
}

type IdempotencyRecord struct {
	RequestID   string
	Operation   string
	Fingerprint string
	StatusCode  int
	Response    json.RawMessage
}

type Commit struct {
	Release          *domain.TestRelease
	ExpectedRevision int64
	Event            domain.AuditEvent
	Idempotency      IdempotencyRecord
	Evidence         []byte
	EvidenceDigest   string
}

type Store interface {
	Create(context.Context, Commit) error
	Load(context.Context, string) (*domain.TestRelease, error)
	List(context.Context, int, int) (domain.Page[domain.TestRelease], error)
	FindActive(context.Context, string, string, string) ([]ReleaseMatch, error)
	Commit(context.Context, Commit) error
	FindIdempotency(context.Context, string) (*IdempotencyRecord, error)
	Audit(context.Context, string, string, int, int) (domain.Page[domain.AuditEvent], error)
	QueryAudit(context.Context, string, AuditFilter) (AuditView, error)
	Evidence(context.Context, string) ([]byte, string, error)
	Close() error
}
