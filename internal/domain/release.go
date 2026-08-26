package domain

import (
	"strings"
	"time"
)

type CreateReleaseInput struct {
	ID               string
	Title            string
	Objective        string
	ModelCode        string
	PlannedCondition string
	Owner            string
}

func NewRelease(in CreateReleaseInput, now time.Time) (*TestRelease, error) {
	in.ID = strings.TrimSpace(in.ID)
	if in.ID == "" {
		return nil, Invalid("id", "不能为空")
	}
	validated := ValidateProfile(in.Title, in.Objective, in.ModelCode, in.PlannedCondition, in.Owner)
	if len(validated.Errors) > 0 {
		return nil, &validated.Errors[0]
	}
	now = now.UTC()
	return &TestRelease{
		ID: in.ID, Title: validated.Title, Objective: validated.Objective, ModelCode: validated.ModelCode,
		PlannedCondition: validated.PlannedCondition, Owner: validated.Owner, Status: StatusDraft,
		Revision: 1, CreatedAt: now, UpdatedAt: now, Channels: []MeasurementChannel{}, Drills: []InterlockDrill{},
	}, nil
}

func (r *TestRelease) EnsureMutable() error {
	if !r.Status.Mutable() {
		return ErrArchived
	}
	return nil
}

func (r *TestRelease) RequireStatus(status Status) error {
	if err := r.EnsureMutable(); err != nil {
		return err
	}
	if r.Status != status {
		return ErrInvalidState
	}
	return nil
}

func (r *TestRelease) UpdateProfile(title, objective, modelCode, condition, owner string) error {
	if err := r.RequireStatus(StatusDraft); err != nil {
		return err
	}
	validated := ValidateProfile(title, objective, modelCode, condition, owner)
	if len(validated.Errors) > 0 {
		return &validated.Errors[0]
	}
	r.Title, r.Objective, r.ModelCode, r.PlannedCondition, r.Owner = validated.Title, validated.Objective, validated.ModelCode, validated.PlannedCondition, validated.Owner
	return nil
}

func (r *TestRelease) GateSummary(now time.Time) []string {
	var gates []string
	if r.Envelope == nil || r.Envelope.EvaluationStatus != "passed" {
		gates = append(gates, "运行边界尚未合格")
	}
	if reasons := VerifyChannels(r.Channels, r.Envelope, now); len(reasons) > 0 {
		gates = append(gates, reasons...)
	}
	if reasons := VerifyDrills(r.Drills); len(reasons) > 0 {
		gates = append(gates, reasons...)
	}
	if r.Witness == nil || r.Witness.Decision != "approved" {
		gates = append(gates, "独立见证尚未签署通过")
	}
	return gates
}
func (r *TestRelease) EnvelopeStatus() string {
	if r.Envelope == nil {
		return "missing"
	}
	return r.Envelope.EvaluationStatus
}

func (r *TestRelease) SafetyRollback(now time.Time) error {
	if r.Status != StatusInterlock && r.Status != StatusWitness && r.Status != StatusAuthorization {
		return ErrInvalidState
	}
	r.InvalidateDownstream(now)
	if r.Witness != nil {
		r.Witness.Decision = "invalidated"
		r.WitnessHistory = append(r.WitnessHistory, *r.Witness)
	}
	r.Status = StatusMeasurement
	return nil
}
