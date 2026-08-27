package evidence_hasher_reuse_test

import (
	"testing"
	"time"

	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
)

func TestEvidenceHasherIsolatedAcrossReleases(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	service := audit.New(func() time.Time { return now })

	first := releasedAggregate(t, "release-first", now)
	firstData, firstDigest, err := service.BuildEvidence(first, nil)
	if err != nil {
		t.Fatalf("构建首个证据包失败: %v", err)
	}
	if err := audit.VerifyEvidence(firstData, firstDigest); err != nil {
		t.Fatalf("首个证据包应可校验: %v", err)
	}

	second := releasedAggregate(t, "release-second", now)
	secondData, secondDigest, err := service.BuildEvidence(second, nil)
	if err != nil {
		t.Fatalf("构建第二个证据包失败: %v", err)
	}
	if err := audit.VerifyEvidence(secondData, secondDigest); err != nil {
		t.Fatalf("同一服务实例中的第二个证据包应使用独立摘要状态: %v", err)
	}
}

func releasedAggregate(t *testing.T, id string, now time.Time) *domain.TestRelease {
	t.Helper()
	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID:               id,
		Title:            "高亚声速模型试验",
		Objective:        "验证安全包摘要隔离",
		ModelCode:        "WT-42",
		PlannedCondition: "马赫数 0.8",
		Owner:            "试验负责人",
	}, now)
	if err != nil {
		t.Fatalf("创建档案失败: %v", err)
	}
	release.Status = domain.StatusReleased
	release.Revision = 9
	release.Witness = &domain.WitnessReview{Decision: "approved", SignedRevision: 8}
	release.Authorization = &domain.Authorization{
		Authorizer:     "授权人",
		Decision:       "approved",
		SignedRevision: 8,
		SignedAt:       now,
	}
	return release
}
