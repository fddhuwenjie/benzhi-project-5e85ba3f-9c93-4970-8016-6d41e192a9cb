package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"windtunnel-release/internal/domain"
	"windtunnel-release/internal/repository"
)

func (s *Server) ListReleases(w http.ResponseWriter, r *http.Request) {
	page, err := s.app.List(r.Context(), queryInt(r, "limit", 25), queryInt(r, "offset", 0))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
func (s *Server) GetRelease(w http.ResponseWriter, r *http.Request) {
	summary, err := s.app.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
func (s *Server) GetAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, err := strictInt(q.Get("limit"), 50, 1, 200)
	if err != nil {
		writeError(w, domain.Invalid("limit", "必须是 1 到 200 的整数"))
		return
	}
	offset, err := strictInt(q.Get("offset"), 0, 0, 1000000)
	if err != nil {
		writeError(w, domain.Invalid("offset", "必须是非负整数"))
		return
	}
	f := repository.AuditFilter{EventType: q.Get("type"), Actor: q.Get("actor"), Limit: limit, Offset: offset}
	f.Role = domain.Role(q.Get("role"))
	f.FromStatus = domain.Status(q.Get("from_status"))
	f.ToStatus = domain.Status(q.Get("to_status"))
	f.RevisionFrom, err = strictInt64(q.Get("revision_from"))
	if err != nil {
		writeError(w, domain.Invalid("revision_from", "必须是正整数"))
		return
	}
	f.RevisionTo, err = strictInt64(q.Get("revision_to"))
	if err != nil {
		writeError(w, domain.Invalid("revision_to", "必须是正整数"))
		return
	}
	if f.RevisionFrom > 0 && f.RevisionTo > 0 && f.RevisionFrom > f.RevisionTo {
		writeError(w, domain.Invalid("revision", "起始 revision 不得大于结束 revision"))
		return
	}
	parseTime := func(key string) (*time.Time, error) {
		v := q.Get(key)
		if v == "" {
			return nil, nil
		}
		t, e := time.Parse(time.RFC3339, v)
		return &t, e
	}
	if q.Get("from") == "" {
		q.Set("from", q.Get("occurred_from"))
	}
	if q.Get("to") == "" {
		q.Set("to", q.Get("occurred_to"))
	}
	f.OccurredFrom, err = parseTime("from")
	if err != nil {
		writeError(w, domain.Invalid("from", "时间必须为 RFC3339"))
		return
	}
	f.OccurredTo, err = parseTime("to")
	if err != nil {
		writeError(w, domain.Invalid("to", "时间必须为 RFC3339"))
		return
	}
	if f.OccurredFrom != nil && f.OccurredTo != nil && f.OccurredTo.Before(*f.OccurredFrom) {
		writeError(w, domain.Invalid("time", "结束时间不得早于开始时间"))
		return
	}
	view, err := s.app.QueryAudit(r.Context(), r.PathValue("id"), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}
func (s *Server) GetChecklist(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.Checklist(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func strictInt(v string, fallback, min, max int) (int, error) {
	if v == "" {
		return fallback, nil
	}
	n, e := strconv.Atoi(v)
	if e != nil || n < min || n > max {
		return 0, fmt.Errorf("invalid")
	}
	return n, nil
}
func strictInt64(v string) (int64, error) {
	if v == "" {
		return 0, nil
	}
	n, e := strconv.ParseInt(v, 10, 64)
	if e != nil || n < 0 {
		return 0, fmt.Errorf("invalid")
	}
	return n, nil
}
func (s *Server) DownloadEvidence(w http.ResponseWriter, r *http.Request) {
	data, digest, err := s.app.Evidence(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	filename := strings.ReplaceAll(r.PathValue("id"), "\"", "")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="evidence-%s.json"`, filename))
	w.Header().Set("X-Evidence-SHA256", digest)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
