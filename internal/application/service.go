package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

type Service struct {
	store         repository.Store
	audit         *audit.Service
	now           func() time.Time
	activeCacheMu sync.RWMutex
	activeCache   map[string][]repository.ReleaseMatch
}

func New(store repository.Store, auditService *audit.Service, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, audit: auditService, now: now, activeCache: make(map[string][]repository.ReleaseMatch)}
}

func NewID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func validateMeta(meta CommandMeta, create bool) error {
	meta.RequestID = strings.TrimSpace(meta.RequestID)
	meta.Actor = strings.TrimSpace(meta.Actor)
	if len(meta.RequestID) < 8 || len(meta.RequestID) > 128 {
		return domain.Invalid("request_id", "长度必须在 8 到 128 个字符之间")
	}
	if meta.Actor == "" {
		return domain.Invalid("actor", "不能为空")
	}
	if create {
		if meta.ExpectedRevision != 0 {
			return domain.ErrConflict
		}
	} else if meta.ExpectedRevision <= 0 {
		return domain.Invalid("expected_revision", "必须大于 0")
	}
	return nil
}

func fingerprint(operation string, input any) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(operation+":"), data...))
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) replay(ctx context.Context, requestID, operation, fp string) (*Result, bool, error) {
	record, err := s.store.FindIdempotency(ctx, requestID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if record.Operation != operation || record.Fingerprint != fp {
		return nil, false, domain.ErrIdempotency
	}
	var result Result
	if err := json.Unmarshal(record.Response, &result); err != nil {
		return nil, false, err
	}
	result.IdempotentReplay = true
	return &result, true, nil
}

func makeRecord(meta CommandMeta, operation, fp string, result *Result) (repository.IdempotencyRecord, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return repository.IdempotencyRecord{}, err
	}
	return repository.IdempotencyRecord{RequestID: meta.RequestID, Operation: operation, Fingerprint: fp, StatusCode: 200, Response: data}, nil
}

type mutation func(*domain.TestRelease) (string, map[string]any, error)

func (s *Service) mutate(ctx context.Context, releaseID, operation string, meta CommandMeta, requiredRole domain.Role, input any, fn mutation) (*Result, error) {
	if err := validateMeta(meta, false); err != nil {
		return nil, err
	}
	if err := domain.RequireRole(meta.Role, requiredRole); err != nil {
		return nil, err
	}
	fp, err := fingerprint(operation, input)
	if err != nil {
		return nil, err
	}
	if result, ok, err := s.replay(ctx, meta.RequestID, operation, fp); err != nil || ok {
		return result, err
	}
	release, err := s.store.Load(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if release.Revision != meta.ExpectedRevision {
		return nil, domain.ErrConflict
	}
	from := release.Status
	eventType, details, err := fn(release)
	if err != nil {
		return nil, err
	}
	release.Revision++
	release.UpdatedAt = s.now().UTC()
	event := s.audit.Event(release.ID, eventType, meta.Actor, meta.Role, from, release.Status, release.Revision, details)
	result := &Result{Release: release, Event: event}
	record, err := makeRecord(meta, operation, fp, result)
	if err != nil {
		return nil, err
	}
	err = s.store.Commit(ctx, repository.Commit{Release: release, ExpectedRevision: meta.ExpectedRevision, Event: event, Idempotency: record})
	if err != nil {
		if replayed, ok, replayErr := s.replay(ctx, meta.RequestID, operation, fp); replayErr == nil && ok {
			return replayed, nil
		}
		return nil, err
	}
	return result, nil
}
