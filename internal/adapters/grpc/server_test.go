package grpc

import (
	"context"
	"testing"

	searchv1 "github.com/Karambollla/wb-search/api/proto/searchv1"
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

func TestGetTop(t *testing.T) {
	server := NewServer(&stubService{top: []core.TopItem{{Query: "носки", Count: 3}}})
	resp, err := server.GetTop(context.Background(), &searchv1.GetTopRequest{Limit: 10})
	if err != nil {
		t.Fatalf("GetTop() error = %v", err)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetQuery() != "носки" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestAddStopWord(t *testing.T) {
	service := &stubService{}
	server := NewServer(service)
	resp, err := server.AddStopWord(context.Background(), &searchv1.StopWordRequest{Term: " Spam "})
	if err != nil {
		t.Fatalf("AddStopWord() error = %v", err)
	}
	if resp.GetTerm() != "spam" {
		t.Fatalf("unexpected term: %q", resp.GetTerm())
	}
}
