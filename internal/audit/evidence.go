package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"windtunnel-release/internal/domain"
)

type EvidencePackage struct {
	SchemaVersion    string                      `json:"schema_version"`
	ReleaseID        string                      `json:"release_id"`
	ReleasedRevision int64                       `json:"released_revision"`
	Profile          EvidenceProfile             `json:"profile"`
	Envelope         *domain.OperatingEnvelope   `json:"envelope"`
	Channels         []domain.MeasurementChannel `json:"channels"`
	Drills           []domain.InterlockDrill     `json:"drills"`
	Witness          *domain.WitnessReview       `json:"witness"`
	Authorization    *domain.Authorization       `json:"authorization"`
	Audit            []domain.AuditEvent         `json:"audit"`
	GeneratedAt      time.Time                   `json:"generated_at"`
}

type EvidenceProfile struct {
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	ModelCode        string `json:"model_code"`
	PlannedCondition string `json:"planned_condition"`
	Owner            string `json:"owner"`
}

func (s *Service) BuildEvidence(release *domain.TestRelease, events []domain.AuditEvent) ([]byte, string, error) {
	if release.Status != domain.StatusReleased || release.Authorization == nil || release.Witness == nil {
		return nil, "", domain.ErrInvalidState
	}
	if release.Authorization.SignedRevision+1 != release.Revision {
		return nil, "", errors.New("授权签署修订与封存修订不一致")
	}
	pkg := EvidencePackage{
		SchemaVersion: "wind-tunnel-release-evidence-v1", ReleaseID: release.ID, ReleasedRevision: release.Revision,
		Profile:  EvidenceProfile{Title: release.Title, Objective: release.Objective, ModelCode: release.ModelCode, PlannedCondition: release.PlannedCondition, Owner: release.Owner},
		Envelope: release.Envelope, Channels: release.Channels, Drills: release.Drills, Witness: release.Witness,
		Authorization: release.Authorization, Audit: events, GeneratedAt: s.now().UTC(),
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if _, err := s.evidenceHash.Write(data); err != nil {
		return nil, "", err
	}
	return data, hex.EncodeToString(s.evidenceHash.Sum(nil)), nil
}

func VerifyEvidence(data []byte, digest string) error {
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != digest {
		return errors.New("证据包摘要校验失败")
	}
	var pkg EvidencePackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}
	if pkg.SchemaVersion != "wind-tunnel-release-evidence-v1" || pkg.ReleaseID == "" {
		return errors.New("证据包结构无效")
	}
	if pkg.Authorization == nil || pkg.ReleasedRevision != pkg.Authorization.SignedRevision+1 {
		return errors.New("证据包签署修订无效")
	}
	return nil
}
