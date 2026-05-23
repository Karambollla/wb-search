package core

import "context"

type EventIngestor interface {
	Ingest(ctx context.Context, event SearchEvent) error
}

type TopReader interface {
	GetTop(ctx context.Context, limit int) ([]TopItem, error)
}

type StopListManager interface {
	ListStopWords(ctx context.Context) ([]string, error)
	AddStopWord(ctx context.Context, term string) error
	DeleteStopWord(ctx context.Context, term string) error
}

type TopService interface {
	EventIngestor
	TopReader
	StopListManager
	RebuildTop(ctx context.Context) error
}
