package core

import (
	"sync"
	"time"
)

type FraudGuard struct {
	mu       sync.Mutex
	ttl      time.Duration
	seenTill map[string]time.Time
}

func NewFraudGuard(ttl time.Duration) *FraudGuard {
	return &FraudGuard{
		ttl:      ttl,
		seenTill: make(map[string]time.Time),
	}
}

func (g *FraudGuard) Allow(event SearchEvent, now time.Time) bool {
	actor := event.UserID
	if actor == "" {
		actor = event.SessionID
	}
	if actor == "" {
		actor = event.IP
	}
	if actor == "" || event.Query == "" || g.ttl <= 0 {
		return true
	}

	key := event.Query + "\x00" + actor

	g.mu.Lock()
	defer g.mu.Unlock()

	if expiresAt, ok := g.seenTill[key]; ok && expiresAt.After(now) {
		return false
	}

	g.seenTill[key] = now.Add(g.ttl)
	if len(g.seenTill)%1024 == 0 {
		g.cleanupLocked(now)
	}
	return true
}

func (g *FraudGuard) cleanupLocked(now time.Time) {
	for key, expiresAt := range g.seenTill {
		if !expiresAt.After(now) {
			delete(g.seenTill, key)
		}
	}
}
