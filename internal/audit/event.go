package audit

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"time"

	"windtunnel-release/internal/domain"
)

type Clock func() time.Time

type Service struct {
	now          Clock
	evidenceHash hash.Hash
}

func New(now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		now:          now,
		evidenceHash: sha256.New(),
	}
}

func (s *Service) Event(releaseID, eventType, actor string, role domain.Role, from, to domain.Status, revision int64, details map[string]any) domain.AuditEvent {
	if details == nil {
		details = map[string]any{}
	}
	return domain.AuditEvent{
		ID: newID("evt"), ReleaseID: releaseID, Type: eventType, Actor: actor, Role: role,
		FromStatus: from, ToStatus: to, Revision: revision, OccurredAt: s.now().UTC(), Details: details,
	}
}

func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(b)
}
