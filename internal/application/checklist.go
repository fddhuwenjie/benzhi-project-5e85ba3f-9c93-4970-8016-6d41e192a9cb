package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func (s *Service) Checklist(ctx context.Context, id string) (Checklist, error) {
	r, err := s.store.Load(ctx, id)
	if err != nil {
		return Checklist{}, err
	}
	channelStates := make([]map[string]any, 0, len(r.Channels))
	for _, c := range r.Channels {
		channelStates = append(channelStates, map[string]any{"id": c.ID, "type": c.ChannelType, "status": c.VerificationStatus, "range_min": c.RangeMin, "range_max": c.RangeMax})
	}
	items := []map[string]any{{"kind": "profile", "title": r.Title, "model_code": r.ModelCode, "owner": r.Owner}, {"kind": "envelope", "status": r.EnvelopeStatus()}, {"kind": "channels", "count": len(r.Channels), "channels": channelStates}, {"kind": "drills", "count": len(r.Drills)}, {"kind": "witness", "reviewer": func() string {
		if r.Witness == nil {
			return ""
		}
		return r.Witness.Reviewer
	}(), "decision": func() string {
		if r.Witness == nil {
			return "pending"
		}
		return r.Witness.Decision
	}()}, {"kind": "signer", "authorizer": func() string {
		if r.Authorization == nil {
			return ""
		}
		return r.Authorization.Authorizer
	}()}}
	canonical, _ := json.Marshal(items)
	sum := sha256.Sum256(canonical)
	return Checklist{ReleaseID: id, Revision: r.Revision, Items: items, Digest: hex.EncodeToString(sum[:])}, nil
}
