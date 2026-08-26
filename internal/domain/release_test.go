package domain

import (
	"testing"
	"time"
)

func TestEnvelopeBlocksAndPassesDeterministically(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r, err := NewRelease(CreateReleaseInput{ID: "r1", Title: "试验", Objective: "目标", ModelCode: "M1", PlannedCondition: "标准工况", Owner: "负责人"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetEnvelope(OperatingEnvelope{SpeedMin: 10, SpeedMax: 500, AttackAngleMin: -2, AttackAngleMax: 3, LoadLimit: 50, TemperatureLimit: 40}); err == nil {
		t.Fatal("超限边界必须阻断")
	}
	if r.Status != StatusDraft {
		t.Fatalf("阻断后状态变化: %s", r.Status)
	}
	if err := r.SetEnvelope(OperatingEnvelope{SpeedMin: 10, SpeedMax: 200, AttackAngleMin: -2, AttackAngleMax: 3, LoadLimit: 50, TemperatureLimit: 40}); err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusMeasurement || r.Envelope.EvaluationStatus != "passed" {
		t.Fatal("合格边界未推进")
	}
}

func TestWitnessRequiresClosedIssuesAndFixedRevision(t *testing.T) {
	r, _ := NewRelease(CreateReleaseInput{ID: "r2", Title: "试验", Objective: "目标", ModelCode: "M2", PlannedCondition: "标准工况", Owner: "负责人"}, time.Now())
	r.Status = StatusWitness
	r.Revision = 7
	r.Witness = &WitnessReview{ID: "w", Reviewer: "见证员", Issues: []WitnessIssue{{ID: "i", Description: "问题"}}}
	if err := r.SignWitness("见证员", 7, time.Now()); err == nil {
		t.Fatal("未整改问题不应签署")
	}
	r.Witness.Issues[0].Closed = true
	r.Witness.Issues[0].RemediationEvidence = "证据"
	if err := r.SignWitness("见证员", 7, time.Now()); err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusAuthorization || r.Witness.SignedRevision != 8 {
		t.Fatal("见证签署未固定下一修订")
	}
}
