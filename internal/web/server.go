package web

import (
	"embed"
	"io/fs"
	"net/http"

	"windtunnel-release/internal/application"
)

//go:embed static/*
var assets embed.FS

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return securityHeaders(s.mux) }

func (s *Server) routes() {
	static, _ := fs.Sub(assets, "static")
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	s.mux.HandleFunc("GET /", s.WorkbenchPage)
	s.mux.HandleFunc("GET /healthz", s.Health)
	s.mux.HandleFunc("GET /api/releases", s.ListReleases)
	s.mux.HandleFunc("POST /api/releases", s.CreateRelease)
	s.mux.HandleFunc("POST /api/releases/precheck", s.Precheck)
	s.mux.HandleFunc("GET /api/releases/{id}", s.GetRelease)
	s.mux.HandleFunc("PUT /api/releases/{id}/profile", s.UpdateProfile)
	s.mux.HandleFunc("POST /api/releases/{id}/envelope", s.EvaluateEnvelope)
	s.mux.HandleFunc("POST /api/releases/{id}/envelope/trial", s.TrialEnvelope)
	s.mux.HandleFunc("POST /api/releases/{id}/channels", s.PutChannel)
	s.mux.HandleFunc("POST /api/releases/{id}/channels/batch", s.ReplaceChannels)
	s.mux.HandleFunc("POST /api/releases/{id}/channels/confirm", s.ConfirmChannels)
	s.mux.HandleFunc("POST /api/releases/{id}/drills", s.PutDrill)
	s.mux.HandleFunc("POST /api/releases/{id}/drills/confirm", s.ConfirmDrills)
	s.mux.HandleFunc("POST /api/releases/{id}/witness", s.RecordWitness)
	s.mux.HandleFunc("POST /api/releases/{id}/witness/remediation", s.RemediateIssue)
	s.mux.HandleFunc("POST /api/releases/{id}/witness/issue", s.ResolveIssue)
	s.mux.HandleFunc("POST /api/releases/{id}/safety-rollback", s.Rollback)
	s.mux.HandleFunc("POST /api/releases/{id}/witness/sign", s.SignWitness)
	s.mux.HandleFunc("POST /api/releases/{id}/authorize", s.AuthorizeRelease)
	s.mux.HandleFunc("GET /api/releases/{id}/checklist", s.GetChecklist)
	s.mux.HandleFunc("GET /api/releases/{id}/audit", s.GetAudit)
	s.mux.HandleFunc("GET /api/releases/{id}/evidence", s.DownloadEvidence)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
