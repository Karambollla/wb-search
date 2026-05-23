package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/Karambollla/wb-search/internal/core"
)

type EventPayload struct {
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	IP        string    `json:"ip,omitempty"`
	Source    string    `json:"source,omitempty"`
}

type Subscriber struct {
	conn    *nats.Conn
	sub     *nats.Subscription
	service core.EventIngestor
	log     *slog.Logger
}

func NewSubscriber(url, subject string, service core.EventIngestor, log *slog.Logger) (*Subscriber, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	sub, err := conn.SubscribeSync(subject)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("subscribe nats: %w", err)
	}
	return &Subscriber{conn: conn, sub: sub, service: service, log: log}, nil
}

func (s *Subscriber) Start(ctx context.Context) {
	for {
		msg, err := s.sub.NextMsgWithContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.log.Warn("failed to read nats message", "error", err)
			continue
		}

		var payload EventPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			s.log.Warn("failed to decode search event", "error", err)
			continue
		}

		err = s.service.Ingest(ctx, core.SearchEvent{
			Query:     payload.Query,
			Timestamp: payload.Timestamp,
			UserID:    payload.UserID,
			SessionID: payload.SessionID,
			IP:        payload.IP,
			Source:    payload.Source,
		})
		if err != nil && err != core.ErrEmptyQuery {
			s.log.Warn("failed to ingest search event", "error", err)
		}
	}
}

func (s *Subscriber) Close() error {
	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			return err
		}
	}
	if s.conn != nil {
		if err := s.conn.Drain(); err != nil {
			s.conn.Close()
			return err
		}
		s.conn.Close()
	}
	return nil
}
