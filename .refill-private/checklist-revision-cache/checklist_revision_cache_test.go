package checklistrevisioncache_test

import (
	"context"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func TestChecklistCacheTracksCommittedRevision(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	store, err := repository.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID:               "release-checklist-cache",
		Title:            "缓存修订核验",
		Objective:        "验证授权清单绑定已提交修订",
		ModelCode:        "CACHE-REV-01",
		PlannedCondition: "Ma 0.5 / alpha 3 deg",
		Owner:            "试验负责人",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	release.Status = domain.StatusWitness
	release.Revision = 7
	release.Witness = &domain.WitnessReview{
		ID:        "review-cache",
		ReleaseID: release.ID,
		Reviewer:  "安全见证员",
		Decision:  "pending",
		Issues:    []domain.WitnessIssue{},
	}
	if err := store.Create(context.Background(), repository.Commit{
		Release: release,
		Event: domain.AuditEvent{
			ID: "event-cache", ReleaseID: release.ID, Type: "witness.reviewed",
			Actor: "安全见证员", Role: domain.RoleWitness, ToStatus: domain.StatusWitness,
			Revision: release.Revision, OccurredAt: now, Details: map[string]any{},
		},
		Idempotency: repository.IdempotencyRecord{RequestID: "setup-cache-release"},
	}); err != nil {
		t.Fatal(err)
	}

	service := application.New(store, audit.New(func() time.Time { return now }), func() time.Time { return now })
	before, err := service.Checklist(context.Background(), release.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Revision != 7 {
		t.Fatalf("初始清单 revision = %d，期望 7", before.Revision)
	}

	_, err = service.SignWitness(context.Background(), release.ID, application.WitnessSignInput{
		CommandMeta: application.CommandMeta{
			RequestID: "sign-cache-revision", ExpectedRevision: 7,
			Actor: "安全见证员", Role: domain.RoleWitness,
		},
		Reviewer: "安全见证员", SignedRevision: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	committed, err := service.Get(context.Background(), release.ID)
	if err != nil {
		t.Fatal(err)
	}
	if committed.Release.Revision != 8 || committed.Release.Status != domain.StatusAuthorization {
		t.Fatalf("见证签署未提交预期状态: revision=%d status=%s", committed.Release.Revision, committed.Release.Status)
	}

	after, err := service.Checklist(context.Background(), release.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != committed.Release.Revision {
		t.Fatalf("清单缓存仍返回 revision %d，提交后的 revision 为 %d", after.Revision, committed.Release.Revision)
	}
	if after.Digest == before.Digest {
		t.Fatal("见证签署后清单摘要仍为旧 revision 的摘要")
	}
}
