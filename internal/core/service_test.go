package core

import (
	"context"
	"strconv"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func TestNormalizeQuery(t *testing.T) {
	got := NormalizeQuery("  Купить   НОСКИ  ")
	if got != "купить носки" {
		t.Fatalf("NormalizeQuery() = %q", got)
	}
}

func TestBucketedCounterEvictsOldBuckets(t *testing.T) {
	counter := NewBucketedCounter(10*time.Second, 5*time.Second)
	now := time.Unix(100, 0)

	counter.Add("fresh", now)
	counter.Add("also fresh", now.Add(-5*time.Second))
	counter.Add("old", now.Add(-11*time.Second))

	snapshot := counter.Snapshot(now)
	if snapshot["fresh"] != 1 {
		t.Fatalf("fresh count = %d", snapshot["fresh"])
	}
	if snapshot["also fresh"] != 1 {
		t.Fatalf("also fresh count = %d", snapshot["also fresh"])
	}
	if snapshot["old"] != 0 {
		t.Fatalf("old query should be evicted, got %d", snapshot["old"])
	}
}

func TestServiceBuildsTopAndAppliesStopList(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	svc := NewService(clock, ServiceConfig{
		Window:         time.Minute,
		BucketDuration: time.Second,
		FraudTTL:       time.Second,
		MaxTopSize:     10,
	})

	events := []SearchEvent{
		{Query: "iPhone", UserID: "u1"},
		{Query: "iphone", UserID: "u2"},
		{Query: "носки", UserID: "u3"},
	}
	for _, event := range events {
		if err := svc.Ingest(context.Background(), event); err != nil {
			t.Fatalf("Ingest() error = %v", err)
		}
	}
	if err := svc.RebuildTop(context.Background()); err != nil {
		t.Fatalf("RebuildTop() error = %v", err)
	}

	top, err := svc.GetTop(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTop() error = %v", err)
	}
	if len(top) != 2 || top[0].Query != "iphone" || top[0].Count != 2 {
		t.Fatalf("unexpected top: %#v", top)
	}

	if err := svc.AddStopWord(context.Background(), "iphone"); err != nil {
		t.Fatalf("AddStopWord() error = %v", err)
	}
	top, err = svc.GetTop(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTop() error = %v", err)
	}
	if len(top) != 1 || top[0].Query != "носки" {
		t.Fatalf("stop-list was not applied to cache: %#v", top)
	}
}

func TestServiceDeduplicatesSameActorQuery(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	svc := NewService(clock, ServiceConfig{
		Window:         time.Minute,
		BucketDuration: time.Second,
		FraudTTL:       time.Minute,
		MaxTopSize:     10,
	})

	event := SearchEvent{Query: "ботинки", UserID: "user-1", Timestamp: clock.now}
	if err := svc.Ingest(context.Background(), event); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if err := svc.Ingest(context.Background(), event); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if err := svc.RebuildTop(context.Background()); err != nil {
		t.Fatalf("RebuildTop() error = %v", err)
	}

	top, err := svc.GetTop(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTop() error = %v", err)
	}
	if len(top) != 1 || top[0].Count != 1 {
		t.Fatalf("expected one counted event after dedup, got %#v", top)
	}
}

func TestServiceDeduplicatesSameUserAcrossDifferentSessions(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	svc := NewService(clock, ServiceConfig{
		Window:         time.Minute,
		BucketDuration: time.Second,
		FraudTTL:       time.Minute,
		MaxTopSize:     10,
	})

	events := []SearchEvent{
		{Query: "куртка", UserID: "user-1", SessionID: "session-1", Timestamp: clock.now},
		{Query: "куртка", UserID: "user-1", SessionID: "session-2", Timestamp: clock.now.Add(time.Second)},
	}
	for _, event := range events {
		if err := svc.Ingest(context.Background(), event); err != nil {
			t.Fatalf("Ingest() error = %v", err)
		}
	}
	if err := svc.RebuildTop(context.Background()); err != nil {
		t.Fatalf("RebuildTop() error = %v", err)
	}

	top, err := svc.GetTop(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTop() error = %v", err)
	}
	if len(top) != 1 || top[0].Count != 1 {
		t.Fatalf("expected same user across sessions to be counted once, got %#v", top)
	}
}

func BenchmarkServiceIngest(b *testing.B) {
	svc := NewService(RealClock{}, ServiceConfig{})
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = svc.Ingest(ctx, SearchEvent{
			Query:  "кроссовки",
			UserID: "bench-user-" + strconv.Itoa(i),
		})
	}
}

func BenchmarkServiceGetTop(b *testing.B) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	svc := NewService(clock, ServiceConfig{})
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_ = svc.Ingest(ctx, SearchEvent{Query: "query", UserID: strconv.Itoa(i)})
	}
	_ = svc.RebuildTop(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetTop(ctx, 10)
	}
}
