package application

import (
	"context"
	"windtunnel-release/internal/domain"
)

func (s *Service) Precheck(ctx context.Context, in PrecheckInput) (*PrecheckResult, error) {
	p := domain.ValidateProfile(in.Title, in.Objective, in.ModelCode, in.PlannedCondition, in.Owner)
	dup, err := s.store.FindActive(ctx, in.ModelCode, in.PlannedCondition, in.ReleaseID)
	if err != nil {
		return nil, err
	}
	result := &PrecheckResult{Profile: p, Duplicates: dup}
	if in.ReleaseID != "" {
		r, e := s.store.Load(ctx, in.ReleaseID)
		if e != nil {
			return nil, e
		}
		result.Differences = []domain.FieldDifference{}
		vals := []struct {
			f    string
			o, c any
			d    float64
		}{{"title", r.Title, p.Title, 0}, {"objective", r.Objective, p.Objective, 0}, {"model_code", r.ModelCode, p.ModelCode, 0}, {"planned_condition", r.PlannedCondition, p.PlannedCondition, 0}, {"owner", r.Owner, p.Owner, 0}}
		for _, v := range vals {
			change := "unchanged"
			if v.o != v.c {
				change = "changed"
			}
			result.Differences = append(result.Differences, domain.FieldDifference{Field: v.f, Previous: v.o, Candidate: v.c, Change: change})
		}
	}
	result.CanProceed = len(p.Errors) == 0
	return result, nil
}
