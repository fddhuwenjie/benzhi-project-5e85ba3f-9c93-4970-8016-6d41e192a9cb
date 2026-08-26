package domain

import (
	"encoding/json"
	"time"
)

type TestRelease struct {
	ID               string               `json:"id"`
	Title            string               `json:"title"`
	Objective        string               `json:"objective"`
	ModelCode        string               `json:"model_code"`
	PlannedCondition string               `json:"planned_condition"`
	Owner            string               `json:"owner"`
	Status           Status               `json:"status"`
	Revision         int64                `json:"revision"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	Envelope         *OperatingEnvelope   `json:"envelope,omitempty"`
	Channels         []MeasurementChannel `json:"channels"`
	Drills           []InterlockDrill     `json:"drills"`
	Witness          *WitnessReview       `json:"witness,omitempty"`
	WitnessHistory   []WitnessReview      `json:"witness_history,omitempty"`
	Authorization    *Authorization       `json:"authorization,omitempty"`
	Evidence         *EvidenceRecord      `json:"evidence,omitempty"`
}

type OperatingEnvelope struct {
	ReleaseID        string          `json:"release_id"`
	SpeedMin         float64         `json:"speed_min"`
	SpeedMax         float64         `json:"speed_max"`
	AttackAngleMin   float64         `json:"attack_angle_min"`
	AttackAngleMax   float64         `json:"attack_angle_max"`
	LoadLimit        float64         `json:"load_limit"`
	TemperatureLimit float64         `json:"temperature_limit"`
	EvaluationStatus string          `json:"evaluation_status"`
	Violations       []string        `json:"violations"`
	Checks           []EnvelopeCheck `json:"checks,omitempty"`
	LastTrial        *EnvelopeTrial  `json:"last_trial,omitempty"`
}

type EnvelopeCheck struct {
	Field     string  `json:"field"`
	Rule      string  `json:"rule"`
	Threshold string  `json:"threshold"`
	Candidate float64 `json:"candidate"`
	Margin    float64 `json:"margin"`
	Passed    bool    `json:"passed"`
}

type EnvelopeTrial struct {
	Candidate   OperatingEnvelope `json:"candidate"`
	Compared    []FieldDifference `json:"compared"`
	RangeImpact map[string]string `json:"range_impact"`
	CreatedAt   time.Time         `json:"created_at"`
}

type FieldDifference struct {
	Field     string `json:"field"`
	Previous  any    `json:"previous"`
	Candidate any    `json:"candidate"`
	Change    string `json:"change"`
}

type MeasurementChannel struct {
	ID                 string    `json:"id"`
	ReleaseID          string    `json:"release_id"`
	ChannelType        string    `json:"channel_type"`
	SensorCode         string    `json:"sensor_code"`
	RangeMin           float64   `json:"range_min"`
	RangeMax           float64   `json:"range_max"`
	CalibratedAt       time.Time `json:"calibrated_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	EvidenceDigest     string    `json:"evidence_digest"`
	VerificationStatus string    `json:"verification_status"`
	RequiredMin        float64   `json:"required_min,omitempty"`
	RequiredMax        float64   `json:"required_max,omitempty"`
	VerificationReason string    `json:"verification_reason,omitempty"`
}

type InterlockDrill struct {
	ID                 string     `json:"id"`
	ReleaseID          string     `json:"release_id"`
	InterlockType      string     `json:"interlock_type"`
	PerformedBy        string     `json:"performed_by"`
	PerformedAt        time.Time  `json:"performed_at"`
	Result             string     `json:"result"`
	ObservedResponseMS int        `json:"observed_response_ms"`
	EvidenceDigest     string     `json:"evidence_digest"`
	AttemptNumber      int        `json:"attempt_number"`
	InvalidatedAt      *time.Time `json:"invalidated_at,omitempty"`
}

type WitnessIssue struct {
	ID                  string              `json:"id"`
	Description         string              `json:"description"`
	RemediationEvidence string              `json:"remediation_evidence,omitempty"`
	Closed              bool                `json:"closed"`
	Status              string              `json:"status,omitempty"`
	EvidenceHistory     []WitnessEvidence   `json:"evidence_history,omitempty"`
	ResolutionHistory   []WitnessResolution `json:"resolution_history,omitempty"`
}

type WitnessEvidence struct {
	Evidence    string    `json:"evidence"`
	SubmittedBy string    `json:"submitted_by"`
	SubmittedAt time.Time `json:"submitted_at"`
}
type WitnessResolution struct {
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

type WitnessReview struct {
	ID                  string         `json:"id"`
	ReleaseID           string         `json:"release_id"`
	Reviewer            string         `json:"reviewer"`
	Observations        string         `json:"observations"`
	Issues              []WitnessIssue `json:"issues"`
	RemediationEvidence []string       `json:"remediation_evidence"`
	Decision            string         `json:"decision"`
	SignedRevision      int64          `json:"signed_revision"`
	SignedAt            *time.Time     `json:"signed_at,omitempty"`
	InvalidatedAt       *time.Time     `json:"invalidated_at,omitempty"`
}

type Authorization struct {
	Authorizer      string    `json:"authorizer"`
	Decision        string    `json:"decision"`
	SignedRevision  int64     `json:"signed_revision"`
	SignedAt        time.Time `json:"signed_at"`
	ChecklistDigest string    `json:"checklist_digest,omitempty"`
}

type EvidenceRecord struct {
	Digest         string          `json:"digest"`
	CanonicalJSON  json.RawMessage `json:"-"`
	CreatedAt      time.Time       `json:"created_at"`
	SignedRevision int64           `json:"signed_revision"`
}

type AuditEvent struct {
	ID         string         `json:"id"`
	ReleaseID  string         `json:"release_id"`
	Type       string         `json:"type"`
	Actor      string         `json:"actor"`
	Role       Role           `json:"role"`
	FromStatus Status         `json:"from_status"`
	ToStatus   Status         `json:"to_status"`
	Revision   int64          `json:"revision"`
	OccurredAt time.Time      `json:"occurred_at"`
	Details    map[string]any `json:"details"`
}

type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
