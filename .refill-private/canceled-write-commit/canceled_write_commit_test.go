package canceledwritecommit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

type cancelOnLoadStore struct {
	repository.Store
	release   *domain.TestRelease
	cancel    context.CancelFunc
	committed bool
}

func (s *cancelOnLoadStore) FindIdempotency(ctx context.Context, _ string) (*repository.IdempotencyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, domain.ErrNotFound
}

func (s *cancelOnLoadStore) Load(ctx context.Context, _ string) (*domain.TestRelease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.cancel()
	return s.release, nil
}

func (s *cancelOnLoadStore) Commit(ctx context.Context, _ repository.Commit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.committed = true
	return nil
}

func TestCanceledMutationDoesNotCommit(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)
	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID:               "rel-context",
		Title:            "低速模型试验",
		Objective:        "验证基准气动载荷",
		ModelCode:        "WT-CANCEL-01",
		PlannedCondition: "速度 40m/s，攻角 0 度",
		Owner:            "试验负责人",
	}, now)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	store := &cancelOnLoadStore{release: release, cancel: cancel}
	service := application.New(store, audit.New(func() time.Time { return now }), func() time.Time { return now })
	result, err := service.UpdateProfile(ctx, release.ID, application.ProfileInput{
		CommandMeta: application.CommandMeta{
			RequestID:        "request-context-canceled",
			ExpectedRevision: release.Revision,
			Actor:            "试验负责人",
			Role:             domain.RoleOwner,
		},
		Title:            "低速模型试验修订",
		Objective:        release.Objective,
		ModelCode:        release.ModelCode,
		PlannedCondition: release.PlannedCondition,
		Owner:            release.Owner,
		ConfirmDiff:      true,
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled mutation must return context.Canceled before commit: result=%v err=%v committed=%v", result, err, store.committed)
	}
	if store.committed {
		t.Fatal("canceled mutation reached Commit")
	}
}
