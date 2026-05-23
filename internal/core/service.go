package core

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
	"time"
)

var ErrEmptyQuery = errors.New("empty query")

type ServiceConfig struct {
	Window          time.Duration
	BucketDuration  time.Duration
	FraudTTL        time.Duration
	MaxTopSize      int
	InitialStopList []string
}

type Service struct {
	clock   Clock
	window  *BucketedCounter
	stop    *StopList
	fraud   *FraudGuard
	maxTop  int
	topList atomic.Value // []TopItem
}

func NewService(clock Clock, cfg ServiceConfig) *Service {
	if clock == nil {
		clock = RealClock{}
	}
	if cfg.Window <= 0 {
		cfg.Window = 5 * time.Minute
	}
	if cfg.BucketDuration <= 0 {
		cfg.BucketDuration = 5 * time.Second
	}
	if cfg.FraudTTL <= 0 {
		cfg.FraudTTL = time.Minute
	}
	if cfg.MaxTopSize <= 0 {
		cfg.MaxTopSize = 100
	}

	s := &Service{
		clock:  clock,
		window: NewBucketedCounter(cfg.Window, cfg.BucketDuration),
		stop:   NewStopList(cfg.InitialStopList),
		fraud:  NewFraudGuard(cfg.FraudTTL),
		maxTop: cfg.MaxTopSize,
	}
	s.topList.Store([]TopItem{})
	return s
}

func (s *Service) Ingest(_ context.Context, event SearchEvent) error {
	event.Query = NormalizeQuery(event.Query)
	if event.Query == "" {
		return ErrEmptyQuery
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = s.clock.Now()
	}
	if s.stop.Contains(event.Query) {
		return nil
	}
	if !s.fraud.Allow(event, event.Timestamp) {
		return nil
	}

	s.window.Add(event.Query, event.Timestamp)
	return nil
}

func (s *Service) RebuildTop(_ context.Context) error {
	counts := s.window.Snapshot(s.clock.Now())
	items := make([]TopItem, 0, len(counts))
	for query, count := range counts {
		if count <= 0 || s.stop.Contains(query) {
			continue
		}
		items = append(items, TopItem{Query: query, Count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Query < items[j].Query
		}
		return items[i].Count > items[j].Count
	})

	if len(items) > s.maxTop {
		items = items[:s.maxTop]
	}
	s.topList.Store(items)
	return nil
}

func (s *Service) GetTop(_ context.Context, limit int) ([]TopItem, error) {
	if limit <= 0 || limit > s.maxTop {
		limit = s.maxTop
	}
	items := s.topList.Load().([]TopItem)
	if limit > len(items) {
		limit = len(items)
	}

	result := make([]TopItem, limit)
	copy(result, items[:limit])
	return result, nil
}

func (s *Service) ListStopWords(_ context.Context) ([]string, error) {
	return s.stop.List(), nil
}

func (s *Service) AddStopWord(ctx context.Context, term string) error {
	s.stop.Add(term)
	return s.RebuildTop(ctx)
}

func (s *Service) DeleteStopWord(ctx context.Context, term string) error {
	s.stop.Delete(term)
	return s.RebuildTop(ctx)
}

func (s *Service) StartTopRefresher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	_ = s.RebuildTop(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.RebuildTop(ctx)
		}
	}
}
