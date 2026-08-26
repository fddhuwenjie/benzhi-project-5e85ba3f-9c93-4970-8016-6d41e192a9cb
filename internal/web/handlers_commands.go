package web

import (
	"net/http"

	"windtunnel-release/internal/application"
)

func (s *Server) CreateRelease(w http.ResponseWriter, r *http.Request) {
	var input application.CreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (s *Server) Precheck(w http.ResponseWriter, r *http.Request) {
	var in application.PrecheckInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := s.app.Precheck(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var input application.ProfileInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.UpdateProfile(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) EvaluateEnvelope(w http.ResponseWriter, r *http.Request) {
	var input application.EnvelopeInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.SetEnvelope(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) TrialEnvelope(w http.ResponseWriter, r *http.Request) {
	var in application.EnvelopeTrialInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := s.app.TrialEnvelope(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) PutChannel(w http.ResponseWriter, r *http.Request) {
	var input application.ChannelInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.PutChannel(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) ReplaceChannels(w http.ResponseWriter, r *http.Request) {
	var in application.ChannelBatchInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := s.app.ReplaceChannels(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) ConfirmChannels(w http.ResponseWriter, r *http.Request) {
	var input application.CommandMeta
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.ConfirmChannels(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) PutDrill(w http.ResponseWriter, r *http.Request) {
	var input application.DrillInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.PutDrill(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) ConfirmDrills(w http.ResponseWriter, r *http.Request) {
	var input application.ConfirmDrillsInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.ConfirmDrills(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) RecordWitness(w http.ResponseWriter, r *http.Request) {
	var input application.WitnessInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.RecordWitness(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) RemediateIssue(w http.ResponseWriter, r *http.Request) {
	var input application.RemediationInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.Remediate(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) ResolveIssue(w http.ResponseWriter, r *http.Request) {
	var in application.IssueResolutionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := s.app.ResolveIssue(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) Rollback(w http.ResponseWriter, r *http.Request) {
	var in application.RollbackInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := s.app.Rollback(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) SignWitness(w http.ResponseWriter, r *http.Request) {
	var input application.WitnessSignInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.SignWitness(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) AuthorizeRelease(w http.ResponseWriter, r *http.Request) {
	var input application.AuthorizationInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.app.Authorize(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
