package audit_checks_alias_test

import (
	"context"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func TestAuditProjectionDoesNotMutateStoredChecks(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	store, err := repository.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID: "audit-alias-release", Title: "审计别名试验", Objective: "验证查询只读边界",
		ModelCode: "AUDIT-ALIAS", PlannedCondition: "Ma 0.5", Owner: "试验负责人",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	checks := []domain.ChannelCheck{
		{ID: "ch-torque", ChannelType: "torque", Status: "passed"},
		{ID: "ch-pressure", ChannelType: "pressure", Status: "passed"},
		{ID: "ch-strain", ChannelType: "strain", Status: "passed"},
	}
	event := domain.AuditEvent{
		ID: "evt-channel-batch", ReleaseID: release.ID, Type: "channels.batch_replaced",
		Actor: "测控工程师", Role: domain.RoleEngineer, FromStatus: domain.StatusMeasurement,
		ToStatus: domain.StatusMeasurement, Revision: release.Revision, OccurredAt: now,
		Details: map[string]any{"checks": checks},
	}
	err = store.Create(ctx, repository.Commit{
		Release: release, Event: event,
		Idempotency: repository.IdempotencyRecord{RequestID: "audit-alias-create", Operation: "release.create"},
	})
	if err != nil {
		t.Fatal(err)
	}

	service := application.New(store, audit.New(func() time.Time { return now }), func() time.Time { return now })
	if _, err := service.QueryAudit(ctx, release.ID, repository.AuditFilter{Limit: 50}); err != nil {
		t.Fatal(err)
	}

	stored, err := store.Audit(ctx, release.ID, "", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	storedChecks, ok := stored.Items[0].Details["checks"].([]domain.ChannelCheck)
	if !ok {
		t.Fatalf("仓储审计 checks 类型异常: %T", stored.Items[0].Details["checks"])
	}
	if storedChecks[0].ChannelType != "torque" || storedChecks[1].ChannelType != "pressure" || storedChecks[2].ChannelType != "strain" {
		t.Fatalf("只读审计查询污染了仓储顺序: %s, %s, %s", storedChecks[0].ChannelType, storedChecks[1].ChannelType, storedChecks[2].ChannelType)
	}
}
