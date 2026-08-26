package application

import (
	"context"
	"fmt"
	"strings"

	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Service) PutChannel(ctx context.Context, id string, input ChannelInput) (*Result, error) {
	return s.mutate(ctx, id, "channel.put", input.CommandMeta, domain.RoleEngineer, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.PutChannel(domain.MeasurementChannel{ID: input.ID, ChannelType: input.ChannelType, SensorCode: input.SensorCode, RangeMin: input.RangeMin, RangeMax: input.RangeMax, CalibratedAt: input.CalibratedAt, ExpiresAt: input.ExpiresAt, EvidenceDigest: input.EvidenceDigest}, s.now())
		return "channel.verified", map[string]any{"channel_type": input.ChannelType, "sensor_code": input.SensorCode}, err
	})
}

func (s *Service) ConfirmChannels(ctx context.Context, id string, meta CommandMeta) (*Result, error) {
	return s.mutate(ctx, id, "channels.confirm", meta, domain.RoleEngineer, meta, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.ConfirmChannels(s.now())
		return "channels.confirmed", map[string]any{"channel_count": len(r.Channels)}, err
	})
}

func (s *Service) TrialEnvelope(ctx context.Context, id string, input EnvelopeTrialInput) (*domain.OperatingEnvelope, error) {
	if err := validateMeta(input.CommandMeta, false); err != nil {
		return nil, err
	}
	if err := domain.RequireRole(input.Role, domain.RoleEngineer); err != nil {
		return nil, err
	}
	r, err := s.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Revision != input.ExpectedRevision {
		return nil, domain.ErrConflict
	}
	v := r.TrialEnvelope(domain.OperatingEnvelope{SpeedMin: input.SpeedMin, SpeedMax: input.SpeedMax, AttackAngleMin: input.AttackAngleMin, AttackAngleMax: input.AttackAngleMax, LoadLimit: input.LoadLimit, TemperatureLimit: input.TemperatureLimit}, s.now())
	return &v, nil
}

func (s *Service) ReplaceChannels(ctx context.Context, id string, input ChannelBatchInput) (*Result, error) {
	if err := validateMeta(input.CommandMeta, false); err != nil {
		return nil, err
	}
	if err := domain.RequireRole(input.Role, domain.RoleEngineer); err != nil {
		return nil, err
	}
	if len(input.Channels) > 50 {
		return nil, domain.Invalid("channels", "批次数量不得超过 50")
	}
	fp, err := fingerprint("channels.replace", input)
	if err != nil {
		return nil, err
	}
	if result, ok, e := s.replay(context.WithoutCancel(ctx), input.RequestID, "channels.replace", fp); e != nil || ok {
		return result, e
	}
	r, err := s.store.Load(context.WithoutCancel(ctx), id)
	if err != nil {
		return nil, err
	}
	if r.Revision != input.ExpectedRevision {
		return nil, domain.ErrConflict
	}
	channels := make([]domain.MeasurementChannel, len(input.Channels))
	for i, c := range input.Channels {
		channels[i] = domain.MeasurementChannel{ID: c.ID, ChannelType: c.ChannelType, SensorCode: c.SensorCode, RangeMin: c.RangeMin, RangeMax: c.RangeMax, CalibratedAt: c.CalibratedAt, ExpiresAt: c.ExpiresAt, EvidenceDigest: c.EvidenceDigest}
	}
	checks, format := r.ReplaceChannels(channels, s.now())
	if len(format) > 0 {
		return nil, &format[0]
	}
	from := r.Status
	r.Revision++
	r.UpdatedAt = s.now().UTC()
	event := s.audit.Event(id, "channels.batch_replaced", input.Actor, input.Role, from, r.Status, r.Revision, map[string]any{"checks": checks})
	result := &Result{Release: r, Event: event}
	record, _ := makeRecord(input.CommandMeta, "channels.replace", fp, result)
	if err := s.store.Commit(context.WithoutCancel(ctx), repository.Commit{Release: r, ExpectedRevision: input.ExpectedRevision, Event: event, Idempotency: record}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) PutDrill(ctx context.Context, id string, input DrillInput) (*Result, error) {
	return s.mutate(ctx, id, "drill.put", input.CommandMeta, domain.RoleEngineer, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.PutDrill(domain.InterlockDrill{ID: input.ID, InterlockType: input.InterlockType, PerformedBy: input.PerformedBy, PerformedAt: input.PerformedAt, Result: input.Result, ObservedResponseMS: input.ObservedResponseMS, EvidenceDigest: input.EvidenceDigest, AttemptNumber: input.AttemptNumber}, s.now())
		return "drill.recorded", map[string]any{"interlock_type": input.InterlockType, "result": input.Result}, err
	})
}

func (s *Service) ConfirmDrills(ctx context.Context, id string, input ConfirmDrillsInput) (*Result, error) {
	return s.mutate(ctx, id, "drills.confirm", input.CommandMeta, domain.RoleEngineer, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		if checks := r.RecheckChannels(s.now()); len(checks) > 0 {
			for _, c := range checks {
				if c.Status != "passed" {
					return "channels.expired", map[string]any{"checks": checks}, domain.Invalid("channels", fmt.Sprintf("通道 %s: %s", c.ChannelType, strings.Join(c.Reasons, "；")))
				}
			}
		}
		err := r.ConfirmDrills(s.now(), input.ReviewID)
		return "drills.confirmed", map[string]any{"snapshot_revision": r.Revision + 1}, err
	})
}
