package web

import (
	"io"
	"net/http"
)

func (s *Server) WorkbenchPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := assets.Open("static/index.html")
	if err != nil {
		http.Error(w, "工作台资源不可用", http.StatusInternalServerError)
		return
	}
	defer data.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, data)
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Health(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
