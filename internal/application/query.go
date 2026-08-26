package application

import (
	"context"
	"time"

	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Service) Get(ctx context.Context, id string) (*Summary, error) {
	release, err := s.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	summary := &Summary{Release: release, StatusLabel: release.Status.Label(), PendingGates: release.GateSummary(s.now())}
	if release.Status == domain.StatusReleased {
		summary.EvidenceURL = "/api/releases/" + release.ID + "/evidence"
		summary.PendingGates = []string{}
	}
	return summary, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) (domain.Page[domain.TestRelease], error) {
	return s.store.List(ctx, limit, offset)
}

func (s *Service) Audit(ctx context.Context, id, eventType string, limit, offset int) (domain.Page[domain.AuditEvent], error) {
	return s.store.Audit(ctx, id, eventType, limit, offset)
}

func (s *Service) QueryAudit(ctx context.Context, id string, filter repository.AuditFilter) (repository.AuditView, error) {
	return s.store.QueryAudit(ctx, id, filter)
}

func (s *Service) Evidence(ctx context.Context, id string) ([]byte, string, error) {
	s.evidenceMu.Lock()
	loader, ok := s.evidenceLoaders[id]
	if !ok {
		loader = newEvidenceLoader(s.store, ctx, id)
		s.evidenceLoaders[id] = loader
	}
	s.evidenceMu.Unlock()

	data, digest, err := loader.Load()
	if err != nil {
		return nil, "", err
	}
	if err := audit.VerifyEvidence(data, digest); err != nil {
		return nil, "", err
	}
	return data, digest, nil
}

func (s *Service) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err := s.store.List(ctx, 1, 0)
	return err
}
