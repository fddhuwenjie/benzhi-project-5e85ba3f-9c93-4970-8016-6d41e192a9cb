package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

var requiredInterlocks = []string{"emergency_stop", "overlimit_cutoff", "data_loss"}

func (r *TestRelease) PutDrill(drill InterlockDrill, clocks ...time.Time) error {
	now := time.Now()
	if len(clocks) > 0 {
		now = clocks[0]
	}
	if err := r.RequireStatus(StatusInterlock); err != nil {
		return err
	}
	drill.ReleaseID = r.ID
	drill.InterlockType = strings.TrimSpace(drill.InterlockType)
	drill.PerformedBy = strings.TrimSpace(drill.PerformedBy)
	drill.EvidenceDigest = strings.TrimSpace(drill.EvidenceDigest)
	if drill.ID == "" || drill.PerformedBy == "" || drill.EvidenceDigest == "" || drill.PerformedAt.IsZero() {
		return Invalid("drill", "演练 ID、执行人、时间和证据摘要不能为空")
	}
	if drill.PerformedAt.After(now.UTC()) {
		return Invalid("performed_at", "不得使用未来执行时间")
	}
	if !contains(requiredInterlocks, drill.InterlockType) {
		return Invalid("interlock_type", "未知联锁类型")
	}
	if drill.Result != "passed" && drill.Result != "failed" {
		return Invalid("result", "结果必须为 passed 或 failed")
	}
	if drill.ObservedResponseMS <= 0 || drill.ObservedResponseMS > 10000 {
		return Invalid("observed_response_ms", "响应时间必须在 1 到 10000 ms 内")
	}
	for _, existing := range r.Drills {
		if existing.ID == drill.ID {
			return Invalid("id", "演练 ID 不得重复")
		}
	}
	maxAttempt := 0
	for _, existing := range r.Drills {
		if existing.InterlockType == drill.InterlockType && existing.AttemptNumber > maxAttempt {
			maxAttempt = existing.AttemptNumber
		}
	}
	if drill.AttemptNumber == 0 {
		drill.AttemptNumber = maxAttempt + 1
	} else if drill.AttemptNumber <= maxAttempt {
		return Invalid("attempt_number", "尝试序号必须递增")
	}
	r.Drills = append(r.Drills, drill)
	sort.Slice(r.Drills, func(i, j int) bool {
		if r.Drills[i].InterlockType == r.Drills[j].InterlockType {
			return r.Drills[i].AttemptNumber < r.Drills[j].AttemptNumber
		}
		return r.Drills[i].InterlockType < r.Drills[j].InterlockType
	})
	return nil
}

func VerifyDrills(drills []InterlockDrill) []string {
	byType := make(map[string]InterlockDrill, len(drills))
	for _, drill := range drills {
		if drill.InvalidatedAt != nil {
			continue
		}
		current, ok := byType[drill.InterlockType]
		if !ok || drill.AttemptNumber > current.AttemptNumber {
			byType[drill.InterlockType] = drill
		}
	}
	var reasons []string
	for _, required := range requiredInterlocks {
		drill, ok := byType[required]
		if !ok {
			reasons = append(reasons, "缺少联锁演练 "+required)
		} else if drill.Result != "passed" {
			reasons = append(reasons, fmt.Sprintf("联锁 %s 演练未通过", required))
		}
	}
	return reasons
}

func (r *TestRelease) InvalidateDownstream(now time.Time) {
	for i := range r.Drills {
		t := now.UTC()
		r.Drills[i].InvalidatedAt = &t
	}
	if r.Witness != nil {
		t := now.UTC()
		r.Witness.InvalidatedAt = &t
	}
	r.Authorization = nil
}

func (r *TestRelease) ConfirmDrills(now time.Time, reviewID string) error {
	if err := r.RequireStatus(StatusInterlock); err != nil {
		return err
	}
	if reasons := VerifyDrills(r.Drills); len(reasons) > 0 {
		return Invalid("drills", strings.Join(reasons, "；"))
	}
	if reviewID == "" {
		return Invalid("review_id", "不能为空")
	}
	if r.Witness != nil {
		r.WitnessHistory = append(r.WitnessHistory, *r.Witness)
	}
	r.Witness = &WitnessReview{ID: reviewID, ReleaseID: r.ID, Issues: []WitnessIssue{}, RemediationEvidence: []string{}, Decision: "pending"}
	r.Status = StatusWitness
	_ = now
	return nil
}
