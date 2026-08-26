package domain

import (
	"strings"
	"time"
)

func (r *TestRelease) RecordWitness(reviewer, observations string, issues []WitnessIssue) error {
	if err := r.RequireStatus(StatusWitness); err != nil {
		return err
	}
	if r.Witness == nil {
		return ErrInvalidState
	}
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		return Invalid("reviewer", "不能为空")
	}
	seen := map[string]bool{}
	for i := range issues {
		issues[i].ID = strings.TrimSpace(issues[i].ID)
		issues[i].Description = strings.TrimSpace(issues[i].Description)
		if issues[i].ID == "" || issues[i].Description == "" {
			return Invalid("issues", "问题 ID 和描述不能为空")
		}
		if seen[issues[i].ID] {
			return Invalid("issues", "问题 ID 不得重复")
		}
		seen[issues[i].ID] = true
		issues[i].Closed = false
		issues[i].Status = "pending_remediation"
		issues[i].RemediationEvidence = ""
		issues[i].EvidenceHistory = nil
		issues[i].ResolutionHistory = nil
	}
	r.Witness.Reviewer = reviewer
	r.Witness.Observations = strings.TrimSpace(observations)
	r.Witness.Issues = issues
	r.Witness.Decision = "pending"
	return nil
}

func (r *TestRelease) RemediateIssue(issueID, evidence string, args ...any) error {
	actor := "负责人"
	now := time.Now()
	if len(args) > 0 {
		if v, ok := args[0].(string); ok {
			actor = v
		}
	}
	if len(args) > 1 {
		if v, ok := args[1].(time.Time); ok {
			now = v
		}
	}
	if err := r.RequireStatus(StatusWitness); err != nil {
		return err
	}
	if r.Witness == nil {
		return ErrInvalidState
	}
	evidence = strings.TrimSpace(evidence)
	if evidence == "" {
		return Invalid("evidence", "整改证据不能为空")
	}
	for i := range r.Witness.Issues {
		if r.Witness.Issues[i].ID == issueID {
			issue := &r.Witness.Issues[i]
			if issue.Status == "closed" {
				return Invalid("issue_id", "问题已关闭，请使用重开")
			}
			issue.RemediationEvidence = evidence
			issue.Closed = false
			issue.Status = "pending_verification"
			issue.EvidenceHistory = append(issue.EvidenceHistory, WitnessEvidence{Evidence: evidence, SubmittedBy: actor, SubmittedAt: now.UTC()})
			r.Witness.RemediationEvidence = append(r.Witness.RemediationEvidence, evidence)
			return nil
		}
	}
	return Invalid("issue_id", "见证问题不存在")
}

func (r *TestRelease) ResolveIssue(issueID, reviewer, action, reason string, now time.Time) error {
	if err := r.RequireStatus(StatusWitness); err != nil {
		return err
	}
	if r.Witness == nil {
		return ErrInvalidState
	}
	if strings.TrimSpace(reviewer) == "" || reviewer != r.Witness.Reviewer {
		return Invalid("reviewer", "必须由原见证员操作")
	}
	for i := range r.Witness.Issues {
		issue := &r.Witness.Issues[i]
		if issue.ID != issueID {
			continue
		}
		if action != "accept" && action != "return" && action != "reopen" {
			return Invalid("action", "不支持的问题操作")
		}
		if action == "accept" {
			if issue.Status != "pending_verification" || issue.RemediationEvidence == "" {
				return Invalid("issue_id", "问题尚未提交整改证据")
			}
			issue.Status = "closed"
			issue.Closed = true
		} else if action == "return" {
			if strings.TrimSpace(reason) == "" {
				return Invalid("reason", "退回原因不能为空")
			}
			issue.Status = "pending_remediation"
			issue.Closed = false
		} else {
			if issue.Status != "closed" {
				return Invalid("issue_id", "只有已关闭问题可以重开")
			}
			if strings.TrimSpace(reason) == "" {
				return Invalid("reason", "重开理由不能为空")
			}
			issue.Status = "pending_remediation"
			issue.Closed = false
		}
		issue.ResolutionHistory = append(issue.ResolutionHistory, WitnessResolution{Action: action, Actor: reviewer, Reason: reason, At: now.UTC()})
		return nil
	}
	return Invalid("issue_id", "见证问题不存在")
}

func (r *TestRelease) SignWitness(reviewer string, signedRevision int64, now time.Time) error {
	if err := r.RequireStatus(StatusWitness); err != nil {
		return err
	}
	if r.Witness == nil || r.Witness.Reviewer == "" {
		return Invalid("witness", "尚未提交见证审查")
	}
	if strings.TrimSpace(reviewer) == "" || reviewer != r.Witness.Reviewer {
		return Invalid("reviewer", "签署人必须与见证员一致")
	}
	for _, issue := range r.Witness.Issues {
		if (issue.Status != "closed" && !(issue.Status == "" && issue.Closed)) || !issue.Closed || issue.RemediationEvidence == "" {
			return Invalid("issues", "所有见证问题必须完成整改并核对证据")
		}
	}
	if signedRevision != r.Revision {
		return ErrConflict
	}
	now = now.UTC()
	r.Witness.Decision = "approved"
	r.Witness.SignedRevision = signedRevision + 1
	r.Witness.SignedAt = &now
	r.Status = StatusAuthorization
	return nil
}

func (r *TestRelease) Authorize(authorizer string, signedRevision int64, now time.Time) error {
	if err := r.RequireStatus(StatusAuthorization); err != nil {
		return err
	}
	authorizer = strings.TrimSpace(authorizer)
	if authorizer == "" {
		return Invalid("authorizer", "不能为空")
	}
	if signedRevision != r.Revision {
		return ErrConflict
	}
	if r.Witness == nil || r.Witness.Decision != "approved" || r.Witness.SignedRevision != r.Revision {
		return Invalid("witness", "见证签署修订与当前固定修订不一致")
	}
	now = now.UTC()
	r.Authorization = &Authorization{Authorizer: authorizer, Decision: "approved", SignedRevision: signedRevision, SignedAt: now}
	r.Status = StatusReleased
	return nil
}
