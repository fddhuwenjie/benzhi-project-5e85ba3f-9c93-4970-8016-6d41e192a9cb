package application

import (
	"context"
	"strings"

	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Service) Precheck(ctx context.Context, in PrecheckInput) (*PrecheckResult, error) {
	p := domain.ValidateProfile(in.Title, in.Objective, in.ModelCode, in.PlannedCondition, in.Owner)
	dup, err := s.findActiveCached(ctx, in.ModelCode, in.PlannedCondition, in.ReleaseID)
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

func (s *Service) findActiveCached(ctx context.Context, modelCode, condition, excludeID string) ([]repository.ReleaseMatch, error) {
	key := strings.ToLower(strings.TrimSpace(modelCode)) + "\x00" + strings.ToLower(strings.TrimSpace(condition)) + "\x00" + excludeID
	s.activeCacheMu.RLock()
	cached, ok := s.activeCache[key]
	s.activeCacheMu.RUnlock()
	if ok {
		return append([]repository.ReleaseMatch(nil), cached...), nil
	}
	matches, err := s.store.FindActive(ctx, modelCode, condition, excludeID)
	if err != nil {
		return nil, err
	}
	s.activeCacheMu.Lock()
	s.activeCache[key] = append([]repository.ReleaseMatch(nil), matches...)
	s.activeCacheMu.Unlock()
	return matches, nil
}
