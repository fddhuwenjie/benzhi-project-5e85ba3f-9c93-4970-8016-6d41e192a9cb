package channelidentitycollision

import (
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func TestPutChannelRejectsCrossIdentityCollision(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	store, err := repository.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	release, err := domain.NewRelease(domain.CreateReleaseInput{ID: "channel-release", Title: "试验", Objective: "目标", ModelCode: "M-CHANNEL", PlannedCondition: "标准", Owner: "负责人"}, now)
	if err != nil {
		t.Fatal(err)
	}
	release.Status = domain.StatusMeasurement
	release.Envelope = &domain.OperatingEnvelope{ReleaseID: release.ID, SpeedMin: 10, SpeedMax: 100, AttackAngleMin: -5, AttackAngleMax: 5, LoadLimit: 40, TemperatureLimit: 50, EvaluationStatus: "passed"}
	release.Channels = []domain.MeasurementChannel{
		{ID: "pressure-id", ReleaseID: release.ID, ChannelType: "pressure", SensorCode: "P-1", RangeMin: 0, RangeMax: 120, CalibratedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), EvidenceDigest: "p-evidence", VerificationStatus: "passed"},
		{ID: "strain-id", ReleaseID: release.ID, ChannelType: "strain", SensorCode: "S-1", RangeMin: -50, RangeMax: 50, CalibratedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), EvidenceDigest: "s-evidence", VerificationStatus: "passed"},
	}
	if err := store.Create(t.Context(), repository.Commit{Release: release, Event: domain.AuditEvent{ReleaseID: release.ID}, Idempotency: repository.IdempotencyRecord{RequestID: "seed-channel-release"}}); err != nil {
		t.Fatal(err)
	}
	app := application.New(store, audit.New(func() time.Time { return now }), func() time.Time { return now })
	_, err = app.PutChannel(t.Context(), release.ID, application.ChannelInput{
		CommandMeta: application.CommandMeta{RequestID: "channel-collision-001", ExpectedRevision: release.Revision, Actor: "工程师", Role: domain.RoleEngineer},
		ID:          "pressure-id", ChannelType: "strain", SensorCode: "S-2", RangeMin: -50, RangeMax: 50, CalibratedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), EvidenceDigest: "replacement-evidence",
	})
	if err == nil {
		t.Fatal("同一请求分别碰撞已有 ID 与已有类型时必须拒绝，不能写入重复类型并丢失原通道")
	}
	stored, loadErr := store.Load(t.Context(), release.ID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(stored.Channels) != 2 || stored.Channels[0].ChannelType != "pressure" || stored.Channels[1].ChannelType != "strain" {
		t.Fatalf("碰撞请求污染了通道集合: %+v", stored.Channels)
	}
}
