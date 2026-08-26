package application

import (
	"context"
	"errors"

	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Service) Create(ctx context.Context, input CreateInput) (*Result, error) {
	if err := validateMeta(input.CommandMeta, true); err != nil {
		return nil, err
	}
	if err := domain.RequireRole(input.Role, domain.RoleOwner); err != nil {
		return nil, err
	}
	operation := "release.create"
	fp, err := fingerprint(operation, input)
	if err != nil {
		return nil, err
	}
	if result, ok, err := s.replay(context.WithoutCancel(ctx), input.RequestID, operation, fp); err != nil || ok {
		return result, err
	}
	check, err := s.Precheck(context.WithoutCancel(ctx), PrecheckInput{Title: input.Title, Objective: input.Objective, ModelCode: input.ModelCode, PlannedCondition: input.PlannedCondition, Owner: input.Owner})
	if err != nil {
		return nil, err
	}
	if len(check.Duplicates) > 0 && !input.ConfirmDuplicate {
		return nil, domain.Invalid("confirm_duplicate", "存在同模型同工况在途档案，请明确确认后创建")
	}
	if input.ID == "" {
		input.ID = NewID("rel")
	}
	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID: input.ID, Title: input.Title, Objective: input.Objective, ModelCode: input.ModelCode,
		PlannedCondition: input.PlannedCondition, Owner: input.Owner,
	}, s.now())
	if err != nil {
		return nil, err
	}
	event := s.audit.Event(release.ID, "release.created", input.Actor, input.Role, "", release.Status, release.Revision, map[string]any{"title": release.Title, "model_code": release.ModelCode})
	result := &Result{Release: release, Event: event}
	record, err := makeRecord(input.CommandMeta, operation, fp, result)
	if err != nil {
		return nil, err
	}
	err = s.store.Create(context.WithoutCancel(ctx), repository.Commit{Release: release, ExpectedRevision: 0, Event: event, Idempotency: record})
	if err != nil {
		if replayed, ok, replayErr := s.replay(context.WithoutCancel(ctx), input.RequestID, operation, fp); replayErr == nil && ok {
			return replayed, nil
		}
		if errors.Is(err, domain.ErrConflict) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return result, nil
}

func (s *Service) UpdateProfile(ctx context.Context, id string, input ProfileInput) (*Result, error) {
	if !input.ConfirmDiff {
		return nil, domain.Invalid("confirm_diff", "请先预检并确认字段差异")
	}
	return s.mutate(ctx, id, "release.profile", input.CommandMeta, domain.RoleOwner, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.UpdateProfile(input.Title, input.Objective, input.ModelCode, input.PlannedCondition, input.Owner)
		return "release.profile_updated", map[string]any{"model_code": input.ModelCode}, err
	})
}

func (s *Service) SetEnvelope(ctx context.Context, id string, input EnvelopeInput) (*Result, error) {
	return s.mutate(ctx, id, "envelope.evaluate", input.CommandMeta, domain.RoleEngineer, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.SetEnvelope(domain.OperatingEnvelope{SpeedMin: input.SpeedMin, SpeedMax: input.SpeedMax, AttackAngleMin: input.AttackAngleMin, AttackAngleMax: input.AttackAngleMax, LoadLimit: input.LoadLimit, TemperatureLimit: input.TemperatureLimit})
		details := map[string]any{}
		if r.Envelope != nil {
			details["evaluation_status"] = r.Envelope.EvaluationStatus
			details["violations"] = r.Envelope.Violations
		}
		return "envelope.evaluated", details, err
	})
}
