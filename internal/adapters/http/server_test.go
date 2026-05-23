package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Karambollla/wb-search/internal/core"
)

type stubService struct {
	top       []core.TopItem
	stopWords []string
}

func (s *stubService) Ingest(context.Context, core.SearchEvent) error { return nil }
func (s *stubService) RebuildTop(context.Context) error               { return nil }
func (s *stubService) GetTop(context.Context, int) ([]core.TopItem, error) {
	return s.top, nil
}
func (s *stubService) ListStopWords(context.Context) ([]string, error) {
	return s.stopWords, nil
}
func (s *stubService) AddStopWord(_ context.Context, term string) error {
	s.stopWords = append(s.stopWords, core.NormalizeQuery(term))
	return nil
}
func (s *stubService) DeleteStopWord(context.Context, string) error { return nil }

func TestTopHandler(t *testing.T) {
	server := NewServer(&stubService{top: []core.TopItem{{Query: "носки", Count: 2}}}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/top?limit=10", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("носки")) {
		t.Fatalf("response does not contain top item: %s", rec.Body.String())
	}
}

func TestAddStopWordHandler(t *testing.T) {
	service := &stubService{}
	server := NewServer(service, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/stoplist", bytes.NewBufferString(`{"term":"  Spam  "}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(service.stopWords) != 1 || service.stopWords[0] != "spam" {
		t.Fatalf("unexpected stop words: %#v", service.stopWords)
	}
}
