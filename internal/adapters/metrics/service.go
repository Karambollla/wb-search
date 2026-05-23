package metrics

import (
	"context"
	"errors"

	"github.com/Karambollla/wb-search/internal/core"
)

type InstrumentedService struct {
	next    core.TopService
	metrics *Metrics
}

func NewInstrumentedService(next core.TopService, metrics *Metrics) *InstrumentedService {
	return &InstrumentedService{next: next, metrics: metrics}
}

func (s *InstrumentedService) Ingest(ctx context.Context, event core.SearchEvent) error {
	err := s.next.Ingest(ctx, event)
	if err != nil {
		if errors.Is(err, core.ErrEmptyQuery) {
			s.metrics.EventsDrop.WithLabelValues("empty_query").Inc()
		}
		return err
	}
	s.metrics.EventsTotal.Inc()
	return nil
}

func (s *InstrumentedService) GetTop(ctx context.Context, limit int) ([]core.TopItem, error) {
	s.metrics.TopReads.Inc()
	return s.next.GetTop(ctx, limit)
}

func (s *InstrumentedService) ListStopWords(ctx context.Context) ([]string, error) {
	return s.next.ListStopWords(ctx)
}

func (s *InstrumentedService) AddStopWord(ctx context.Context, term string) error {
	return s.next.AddStopWord(ctx, term)
}

func (s *InstrumentedService) DeleteStopWord(ctx context.Context, term string) error {
	return s.next.DeleteStopWord(ctx, term)
}

func (s *InstrumentedService) RebuildTop(ctx context.Context) error {
	return s.next.RebuildTop(ctx)
}
