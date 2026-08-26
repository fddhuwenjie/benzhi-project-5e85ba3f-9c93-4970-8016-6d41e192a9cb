package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Service) RecordWitness(ctx context.Context, id string, input WitnessInput) (*Result, error) {
	return s.mutate(ctx, id, "witness.review", input.CommandMeta, domain.RoleWitness, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.RecordWitness(input.Reviewer, input.Observations, input.Issues)
		return "witness.reviewed", map[string]any{"issue_count": len(input.Issues)}, err
	})
}

func (s *Service) Remediate(ctx context.Context, id string, input RemediationInput) (*Result, error) {
	return s.mutate(ctx, id, "witness.remediate", input.CommandMeta, domain.RoleOwner, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.RemediateIssue(input.IssueID, input.Evidence, input.Actor, s.now())
		return "witness.issue_remediated", map[string]any{"issue_id": input.IssueID}, err
	})
}

func (s *Service) ResolveIssue(ctx context.Context, id string, input IssueResolutionInput) (*Result, error) {
	if strings.TrimSpace(input.Actor) != strings.TrimSpace(input.Reviewer) {
		return nil, domain.Invalid("actor", "操作者必须与见证员身份一致")
	}
	return s.mutate(ctx, id, "witness.issue_"+input.Action, input.CommandMeta, domain.RoleWitness, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.ResolveIssue(input.IssueID, input.Reviewer, input.Action, input.Reason, s.now())
		return "witness.issue_" + input.Action, map[string]any{"issue_id": input.IssueID, "reason": input.Reason}, err
	})
}

func (s *Service) Rollback(ctx context.Context, id string, input RollbackInput) (*Result, error) {
	return s.mutate(ctx, id, "release.safety_rollback", input.CommandMeta, domain.RoleEngineer, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		err := r.SafetyRollback(s.now())
		return "release.safety_rollback", map[string]any{"reason": input.Reason}, err
	})
}

func (s *Service) SignWitness(ctx context.Context, id string, input WitnessSignInput) (*Result, error) {
	return s.mutate(ctx, id, "witness.sign", input.CommandMeta, domain.RoleWitness, input, func(r *domain.TestRelease) (string, map[string]any, error) {
		for _, c := range r.RecheckChannels(s.now()) {
			if c.Status != "passed" {
				return "channels.expired", map[string]any{"channel": c}, domain.Invalid("channels", fmt.Sprintf("通道 %s: %s", c.ChannelType, strings.Join(c.Reasons, "；")))
			}
		}
		err := r.SignWitness(input.Reviewer, input.SignedRevision, s.now())
		return "witness.signed", map[string]any{"signed_revision": input.SignedRevision}, err
	})
}

func (s *Service) Authorize(ctx context.Context, id string, input AuthorizationInput) (*Result, error) {
	if err := validateMeta(input.CommandMeta, false); err != nil {
		return nil, err
	}
	if err := domain.RequireRole(input.Role, domain.RoleAuthorizer); err != nil {
		return nil, err
	}
	operation := "release.authorize"
	fp, err := fingerprint(operation, input)
	if err != nil {
		return nil, err
	}
	if result, ok, err := s.replay(context.WithoutCancel(ctx), input.RequestID, operation, fp); err != nil || ok {
		return result, err
	}
	release, err := s.store.Load(context.WithoutCancel(ctx), id)
	if err != nil {
		return nil, err
	}
	if release.Revision != input.ExpectedRevision {
		return nil, domain.ErrConflict
	}
	for _, c := range release.RecheckChannels(s.now()) {
		if c.Status != "passed" {
			return nil, domain.Invalid("channels", fmt.Sprintf("通道 %s: %s", c.ChannelType, strings.Join(c.Reasons, "；")))
		}
	}
	check, err := s.Checklist(context.WithoutCancel(ctx), id)
	if err != nil {
		return nil, err
	}
	if input.ChecklistDigest == "" {
		return nil, domain.Invalid("checklist_digest", "必须确认授权前固定清单摘要")
	}
	if input.ChecklistDigest != check.Digest {
		return nil, domain.ErrConflict
	}
	if input.SignedRevision != check.Revision {
		return nil, domain.ErrConflict
	}
	from := release.Status
	if err := release.Authorize(input.Authorizer, input.SignedRevision, s.now()); err != nil {
		return nil, err
	}
	release.Revision++
	release.UpdatedAt = s.now().UTC()
	event := s.audit.Event(id, "release.authorized", input.Actor, input.Role, from, release.Status, release.Revision, map[string]any{"signed_revision": input.SignedRevision, "checklist_digest": check.Digest})
	page, err := s.store.Audit(context.WithoutCancel(ctx), id, "", 10000, 0)
	if err != nil {
		return nil, err
	}
	events := append(page.Items, event)
	if release.Authorization != nil {
		release.Authorization.ChecklistDigest = check.Digest
	}
	evidence, digest, err := s.audit.BuildEvidence(release, events)
	if err != nil {
		return nil, err
	}
	event.Details["evidence_digest"] = digest
	release.Evidence = &domain.EvidenceRecord{Digest: digest, CanonicalJSON: json.RawMessage(evidence), CreatedAt: s.now().UTC(), SignedRevision: input.SignedRevision}
	result := &Result{Release: release, Event: event}
	record, err := makeRecord(input.CommandMeta, operation, fp, result)
	if err != nil {
		return nil, err
	}
	err = s.store.Commit(context.WithoutCancel(ctx), repository.Commit{Release: release, ExpectedRevision: input.ExpectedRevision, Event: event, Idempotency: record, Evidence: evidence, EvidenceDigest: digest})
	if err != nil {
		return nil, err
	}
	return result, nil
}
