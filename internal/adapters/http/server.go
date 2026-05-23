package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Karambollla/wb-search/internal/core"
)

type Metrics interface {
	ObserveTopRead()
}

type Server struct {
	service core.TopService
	mux     *http.ServeMux
}

func NewServer(service core.TopService, metricsHandler http.Handler) *Server {
	s := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}
	if metricsHandler == nil {
		metricsHandler = promhttp.Handler()
	}
	s.routes(metricsHandler)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes(metricsHandler http.Handler) {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.Handle("GET /metrics", metricsHandler)
	s.mux.HandleFunc("GET /v1/top", s.top)
	s.mux.HandleFunc("GET /v1/stoplist", s.listStopWords)
	s.mux.HandleFunc("POST /v1/stoplist", s.addStopWord)
	s.mux.HandleFunc("DELETE /v1/stoplist/{term}", s.deleteStopWord)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) top(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}

	items, err := s.service.GetTop(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) listStopWords(w http.ResponseWriter, r *http.Request) {
	terms, err := s.service.ListStopWords(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": terms})
}

type stopWordRequest struct {
	Term string `json:"term"`
}

func (s *Server) addStopWord(w http.ResponseWriter, r *http.Request) {
	var req stopWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Term) == "" {
		writeError(w, http.StatusBadRequest, "term is required")
		return
	}
	if err := s.service.AddStopWord(r.Context(), req.Term); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"term": core.NormalizeQuery(req.Term)})
}

func (s *Server) deleteStopWord(w http.ResponseWriter, r *http.Request) {
	term := r.PathValue("term")
	if strings.TrimSpace(term) == "" {
		writeError(w, http.StatusBadRequest, "term is required")
		return
	}
	if err := s.service.DeleteStopWord(r.Context(), term); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"term": core.NormalizeQuery(term)})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if message == "" {
		message = http.StatusText(status)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func StatusCode(err error) int {
	if errors.Is(err, core.ErrEmptyQuery) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
