package repository

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"

	"windtunnel-release/internal/domain"
)

type SQLiteStore struct {
	mu         sync.RWMutex
	path       string
	releases   map[string]*domain.TestRelease
	events     map[string][]domain.AuditEvent
	idempotent map[string]IdempotencyRecord
	evidence   map[string]evidenceItem
}
type evidenceItem struct {
	Data   []byte `json:"data"`
	Digest string `json:"digest"`
}
type snapshot struct {
	Releases   map[string]*domain.TestRelease `json:"releases"`
	Events     map[string][]domain.AuditEvent `json:"events"`
	Idempotent map[string]IdempotencyRecord   `json:"idempotent"`
	Evidence   map[string]evidenceItem        `json:"evidence"`
}

func Open(path string) (*SQLiteStore, error) {
	if path == "" {
		path = "windtunnel.db.json"
	}
	s := &SQLiteStore{path: path, releases: map[string]*domain.TestRelease{}, events: map[string][]domain.AuditEvent{}, idempotent: map[string]IdempotencyRecord{}, evidence: map[string]evidenceItem{}}
	if path == ":memory:" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err == nil {
		var snap snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			return nil, err
		}
		if snap.Releases != nil {
			s.releases = snap.Releases
		}
		if snap.Events != nil {
			s.events = snap.Events
		}
		if snap.Idempotent != nil {
			s.idempotent = snap.Idempotent
		}
		if snap.Evidence != nil {
			s.evidence = snap.Evidence
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}
func (s *SQLiteStore) Close() error { return nil }
func (s *SQLiteStore) persistLocked() error {
	if s.path == ":memory:" {
		return nil
	}
	data, err := json.Marshal(snapshot{s.releases, s.events, s.idempotent, s.evidence})
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
func cloneRelease(r *domain.TestRelease) *domain.TestRelease {
	if r == nil {
		return nil
	}
	c := *r
	c.Channels = append([]domain.MeasurementChannel(nil), r.Channels...)
	c.Drills = append([]domain.InterlockDrill(nil), r.Drills...)
	if r.Envelope != nil {
		envelope := *r.Envelope
		envelope.Violations = append([]string(nil), r.Envelope.Violations...)
		envelope.Checks = append([]domain.EnvelopeCheck(nil), r.Envelope.Checks...)
		if r.Envelope.LastTrial != nil {
			trial := *r.Envelope.LastTrial
			trial.Compared = append([]domain.FieldDifference(nil), r.Envelope.LastTrial.Compared...)
			trial.RangeImpact = make(map[string]string, len(r.Envelope.LastTrial.RangeImpact))
			for key, value := range r.Envelope.LastTrial.RangeImpact {
				trial.RangeImpact[key] = value
			}
			envelope.LastTrial = &trial
		}
		c.Envelope = &envelope
	}
	c.Witness = cloneWitness(r.Witness)
	c.WitnessHistory = make([]domain.WitnessReview, len(r.WitnessHistory))
	for i := range r.WitnessHistory {
		c.WitnessHistory[i] = *cloneWitness(&r.WitnessHistory[i])
	}
	if r.Authorization != nil {
		authorization := *r.Authorization
		c.Authorization = &authorization
	}
	if r.Evidence != nil {
		evidence := *r.Evidence
		evidence.CanonicalJSON = append(json.RawMessage(nil), r.Evidence.CanonicalJSON...)
		c.Evidence = &evidence
	}
	return &c
}

func cloneWitness(w *domain.WitnessReview) *domain.WitnessReview {
	if w == nil {
		return nil
	}
	cloned := *w
	cloned.RemediationEvidence = append([]string(nil), w.RemediationEvidence...)
	cloned.Issues = make([]domain.WitnessIssue, len(w.Issues))
	for i := range w.Issues {
		issue := w.Issues[i]
		issue.EvidenceHistory = append([]domain.WitnessEvidence(nil), w.Issues[i].EvidenceHistory...)
		issue.ResolutionHistory = append([]domain.WitnessResolution(nil), w.Issues[i].ResolutionHistory...)
		cloned.Issues[i] = issue
	}
	return &cloned
}
func (s *SQLiteStore) Create(ctx context.Context, c Commit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.releases[c.Release.ID]; ok {
		return domain.ErrConflict
	}
	s.releases[c.Release.ID] = cloneRelease(c.Release)
	s.events[c.Release.ID] = append(s.events[c.Release.ID], c.Event)
	s.idempotent[c.Idempotency.RequestID] = c.Idempotency
	return s.persistLocked()
}
func (s *SQLiteStore) Load(ctx context.Context, id string) (*domain.TestRelease, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.releases[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneRelease(r), nil
}
func (s *SQLiteStore) List(ctx context.Context, limit, offset int) (domain.Page[domain.TestRelease], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	items := make([]domain.TestRelease, 0, len(s.releases))
	for _, r := range s.releases {
		items = append(items, *cloneRelease(r))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return domain.Page[domain.TestRelease]{Items: items[offset:end], Total: len(items), Limit: limit, Offset: offset}, nil
}
func (s *SQLiteStore) FindActive(ctx context.Context, modelCode, condition, excludeID string) ([]ReleaseMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	modelCode = strings.TrimSpace(modelCode)
	condition = strings.TrimSpace(condition)
	items := []ReleaseMatch{}
	for _, r := range s.releases {
		if r.ID != excludeID && r.Status != domain.StatusReleased && strings.EqualFold(strings.TrimSpace(r.ModelCode), modelCode) && strings.EqualFold(strings.TrimSpace(r.PlannedCondition), condition) {
			items = append(items, ReleaseMatch{ID: r.ID, Status: r.Status, Owner: r.Owner, UpdatedAt: r.UpdatedAt})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}
func (s *SQLiteStore) Commit(ctx context.Context, c Commit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.releases[c.Release.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if r.Revision != c.ExpectedRevision {
		return domain.ErrConflict
	}
	s.releases[c.Release.ID] = cloneRelease(c.Release)
	s.events[c.Release.ID] = append(s.events[c.Release.ID], c.Event)
	s.idempotent[c.Idempotency.RequestID] = c.Idempotency
	if len(c.Evidence) > 0 {
		s.evidence[c.Release.ID] = evidenceItem{c.Evidence, c.EvidenceDigest}
	}
	return s.persistLocked()
}
func (s *SQLiteStore) FindIdempotency(ctx context.Context, id string) (*IdempotencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.idempotent[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &r, nil
}
func (s *SQLiteStore) Audit(ctx context.Context, id, eventType string, limit, offset int) (domain.Page[domain.AuditEvent], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.events[id]
	filtered := make([]domain.AuditEvent, 0, len(all))
	for _, e := range all {
		if eventType == "" || e.Type == eventType {
			filtered = append(filtered, e)
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return domain.Page[domain.AuditEvent]{Items: filtered[offset:end], Total: len(filtered), Limit: limit, Offset: offset}, nil
}
func (s *SQLiteStore) QueryAudit(ctx context.Context, id string, f AuditFilter) (AuditView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filtered := []domain.AuditEvent{}
	actors := map[string]bool{}
	stages := map[domain.Status]int{}
	failures := 0
	for _, e := range s.events[id] {
		if f.EventType != "" && e.Type != f.EventType {
			continue
		}
		if f.Role != "" && e.Role != f.Role {
			continue
		}
		if f.Actor != "" && e.Actor != f.Actor {
			continue
		}
		if f.RevisionFrom > 0 && e.Revision < f.RevisionFrom {
			continue
		}
		if f.RevisionTo > 0 && e.Revision > f.RevisionTo {
			continue
		}
		if f.FromStatus != "" && e.FromStatus != f.FromStatus {
			continue
		}
		if f.ToStatus != "" && e.ToStatus != f.ToStatus {
			continue
		}
		if f.OccurredFrom != nil && e.OccurredAt.Before(*f.OccurredFrom) {
			continue
		}
		if f.OccurredTo != nil && e.OccurredAt.After(*f.OccurredTo) {
			continue
		}
		filtered = append(filtered, e)
		actors[e.Actor] = true
		stages[e.ToStatus]++
		if strings.Contains(e.Type, "failed") || strings.Contains(e.Type, "return") || strings.Contains(e.Type, "reopened") || strings.Contains(e.Type, "rollback") {
			failures++
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if !filtered[i].OccurredAt.Equal(filtered[j].OccurredAt) {
			return filtered[i].OccurredAt.Before(filtered[j].OccurredAt)
		}
		if filtered[i].Revision != filtered[j].Revision {
			return filtered[i].Revision < filtered[j].Revision
		}
		return filtered[i].ID < filtered[j].ID
	})
	stats := AuditStats{Total: len(filtered), FailureOrReturnCount: failures, DistinctActors: len(actors), Stages: []StageCount{}}
	if len(filtered) > 0 {
		first, last := filtered[0].OccurredAt, filtered[len(filtered)-1].OccurredAt
		stats.FirstOccurredAt = &first
		stats.LastOccurredAt = &last
	}
	order := []domain.Status{domain.StatusDraft, domain.StatusMeasurement, domain.StatusInterlock, domain.StatusWitness, domain.StatusAuthorization, domain.StatusReleased}
	for _, status := range order {
		stats.Stages = append(stats.Stages, StageCount{Status: status, Count: stages[status]})
	}
	if f.Offset > len(filtered) {
		f.Offset = len(filtered)
	}
	end := f.Offset + f.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	items := filtered[f.Offset:end]
	return AuditView{Page: domain.Page[domain.AuditEvent]{Items: items, Total: len(filtered), Limit: f.Limit, Offset: f.Offset}, Stats: stats}, nil
}
func (s *SQLiteStore) Evidence(ctx context.Context, id string) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.evidence[id]
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return append([]byte(nil), e.Data...), e.Digest, nil
}
