package core

import "time"

type SearchEvent struct {
	Query     string
	Timestamp time.Time
	UserID    string
	SessionID string
	IP        string
	Source    string
}

type TopItem struct {
	Query string
	Count int64
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}
