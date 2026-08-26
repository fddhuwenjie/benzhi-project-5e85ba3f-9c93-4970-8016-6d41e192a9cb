package audit

import (
	"testing"
	"time"

	"windtunnel-release/internal/domain"
)

func TestEvidenceDigestVerification(t *testing.T) {
	r, _ := domain.NewRelease(domain.CreateReleaseInput{ID: "r", Title: "试验", Objective: "目标", ModelCode: "M", PlannedCondition: "工况", Owner: "负责人"}, time.Now())
	r.Status = domain.StatusReleased
	r.Revision = 2
	r.Witness = &domain.WitnessReview{Decision: "approved", SignedRevision: 1}
	r.Authorization = &domain.Authorization{Authorizer: "授权人", Decision: "approved", SignedRevision: 1, SignedAt: time.Now()}
	s := New(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })
	data, digest, err := s.BuildEvidence(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyEvidence(data, digest); err != nil {
		t.Fatal(err)
	}
	data[0] = 'x'
	if err := VerifyEvidence(data, digest); err == nil {
		t.Fatal("篡改证据应失败")
	}
}
