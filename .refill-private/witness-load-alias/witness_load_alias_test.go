package witness_load_alias_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"windtunnel-release/internal/application"
	"windtunnel-release/internal/audit"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

var errCommitRejected = errors.New("controlled commit rejection")

type rejectCommitStore struct {
	repository.Store
}

func (s rejectCommitStore) Commit(context.Context, repository.Commit) error {
	return errCommitRejected
}

func TestRejectedIssueResolutionDoesNotMutateStoredWitness(t *testing.T) {
	now := time.Date(2026, 4, 5, 6, 7, 8, 0, time.UTC)
	store, err := repository.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	release, err := domain.NewRelease(domain.CreateReleaseInput{
		ID: "release-alias", Title: "低速段试验", Objective: "确认测量链", ModelCode: "M-ALIAS",
		PlannedCondition: "标准低速工况", Owner: "负责人",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	release.Status = domain.StatusWitness
	release.Revision = 7
	release.Witness = &domain.WitnessReview{
		ID: "review-alias", ReleaseID: release.ID, Reviewer: "见证员", Decision: "pending",
		Issues: []domain.WitnessIssue{{
			ID: "issue-alias", Description: "补充校准记录", Status: "pending_verification",
			RemediationEvidence: "sha256:evidence", Closed: false,
		}},
	}
	if err := store.Create(context.Background(), repository.Commit{
		Release:     release,
		Event:       domain.AuditEvent{ID: "event-created", ReleaseID: release.ID, Type: "fixture.created", Revision: release.Revision},
		Idempotency: repository.IdempotencyRecord{RequestID: "fixture-create", Operation: "fixture.create"},
	}); err != nil {
		t.Fatal(err)
	}

	service := application.New(rejectCommitStore{Store: store}, audit.New(func() time.Time { return now }), func() time.Time { return now })
	_, err = service.ResolveIssue(context.Background(), release.ID, application.IssueResolutionInput{
		CommandMeta: application.CommandMeta{RequestID: "resolve-rejected", ExpectedRevision: 7, Actor: "见证员", Role: domain.RoleWitness},
		IssueID:     "issue-alias", Action: "accept", Reviewer: "见证员",
	})
	if !errors.Is(err, errCommitRejected) {
		t.Fatalf("expected controlled commit rejection, got %v", err)
	}

	stored, err := store.Load(context.Background(), release.ID)
	if err != nil {
		t.Fatal(err)
	}
	issue := stored.Witness.Issues[0]
	if issue.Closed || issue.Status != "pending_verification" || len(issue.ResolutionHistory) != 0 {
		t.Fatalf("rejected mutation leaked through Load alias: closed=%v status=%q history=%d", issue.Closed, issue.Status, len(issue.ResolutionHistory))
	}
	if stored.Revision != 7 {
		t.Fatalf("rejected mutation changed revision: %d", stored.Revision)
	}
}
