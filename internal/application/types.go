package application

import (
	"time"

	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

type CommandMeta struct {
	RequestID        string      `json:"request_id"`
	ExpectedRevision int64       `json:"expected_revision"`
	Actor            string      `json:"actor"`
	Role             domain.Role `json:"role"`
}

type Result struct {
	Release          *domain.TestRelease `json:"release"`
	Event            domain.AuditEvent   `json:"event"`
	IdempotentReplay bool                `json:"idempotent_replay"`
}

type CreateInput struct {
	CommandMeta
	ID               string `json:"id"`
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	ModelCode        string `json:"model_code"`
	PlannedCondition string `json:"planned_condition"`
	Owner            string `json:"owner"`
	ConfirmDuplicate bool   `json:"confirm_duplicate,omitempty"`
}

type ProfileInput struct {
	CommandMeta
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	ModelCode        string `json:"model_code"`
	PlannedCondition string `json:"planned_condition"`
	Owner            string `json:"owner"`
	ConfirmDiff      bool   `json:"confirm_diff,omitempty"`
}

type PrecheckInput struct {
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	ModelCode        string `json:"model_code"`
	PlannedCondition string `json:"planned_condition"`
	Owner            string `json:"owner"`
	ReleaseID        string `json:"release_id,omitempty"`
}
type PrecheckResult struct {
	Profile     domain.ProfileValidation  `json:"profile"`
	Duplicates  []repository.ReleaseMatch `json:"duplicates"`
	Differences []domain.FieldDifference  `json:"differences,omitempty"`
	CanProceed  bool                      `json:"can_proceed"`
}

type EnvelopeInput struct {
	CommandMeta
	SpeedMin         float64 `json:"speed_min"`
	SpeedMax         float64 `json:"speed_max"`
	AttackAngleMin   float64 `json:"attack_angle_min"`
	AttackAngleMax   float64 `json:"attack_angle_max"`
	LoadLimit        float64 `json:"load_limit"`
	TemperatureLimit float64 `json:"temperature_limit"`
}

type ChannelInput struct {
	CommandMeta
	ID             string    `json:"id"`
	ChannelType    string    `json:"channel_type"`
	SensorCode     string    `json:"sensor_code"`
	RangeMin       float64   `json:"range_min"`
	RangeMax       float64   `json:"range_max"`
	CalibratedAt   time.Time `json:"calibrated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	EvidenceDigest string    `json:"evidence_digest"`
}
type ChannelBatchInput struct {
	CommandMeta
	Channels []ChannelInput `json:"channels"`
}
type EnvelopeTrialInput struct {
	CommandMeta
	SpeedMin         float64 `json:"speed_min"`
	SpeedMax         float64 `json:"speed_max"`
	AttackAngleMin   float64 `json:"attack_angle_min"`
	AttackAngleMax   float64 `json:"attack_angle_max"`
	LoadLimit        float64 `json:"load_limit"`
	TemperatureLimit float64 `json:"temperature_limit"`
}

type DrillInput struct {
	CommandMeta
	ID                 string    `json:"id"`
	InterlockType      string    `json:"interlock_type"`
	PerformedBy        string    `json:"performed_by"`
	PerformedAt        time.Time `json:"performed_at"`
	Result             string    `json:"result"`
	ObservedResponseMS int       `json:"observed_response_ms"`
	EvidenceDigest     string    `json:"evidence_digest"`
	AttemptNumber      int       `json:"attempt_number,omitempty"`
}

type ConfirmDrillsInput struct {
	CommandMeta
	ReviewID string `json:"review_id"`
}

type WitnessInput struct {
	CommandMeta
	Reviewer     string                `json:"reviewer"`
	Observations string                `json:"observations"`
	Issues       []domain.WitnessIssue `json:"issues"`
}

type RemediationInput struct {
	CommandMeta
	IssueID  string `json:"issue_id"`
	Evidence string `json:"evidence"`
}
type IssueResolutionInput struct {
	CommandMeta
	IssueID  string `json:"issue_id"`
	Action   string `json:"action"`
	Reason   string `json:"reason,omitempty"`
	Reviewer string `json:"reviewer"`
}
type RollbackInput struct {
	CommandMeta
	Reason string `json:"reason"`
}

type WitnessSignInput struct {
	CommandMeta
	Reviewer       string `json:"reviewer"`
	SignedRevision int64  `json:"signed_revision"`
}

type AuthorizationInput struct {
	CommandMeta
	Authorizer      string `json:"authorizer"`
	SignedRevision  int64  `json:"signed_revision"`
	ChecklistDigest string `json:"checklist_digest"`
}

type Checklist struct {
	ReleaseID string           `json:"release_id"`
	Revision  int64            `json:"revision"`
	Items     []map[string]any `json:"items"`
	Digest    string           `json:"digest"`
}

type Summary struct {
	Release      *domain.TestRelease `json:"release"`
	StatusLabel  string              `json:"status_label"`
	PendingGates []string            `json:"pending_gates"`
	EvidenceURL  string              `json:"evidence_url,omitempty"`
}
