package core

import (
	"sort"
	"sync"
	"sync/atomic"
)

type StopList struct {
	mu    sync.Mutex
	terms atomic.Value // map[string]struct{}
}

func NewStopList(initial []string) *StopList {
	s := &StopList{}
	m := make(map[string]struct{}, len(initial))
	for _, term := range initial {
		if normalized := NormalizeQuery(term); normalized != "" {
			m[normalized] = struct{}{}
		}
	}
	s.terms.Store(m)
	return s
}

func (s *StopList) Contains(term string) bool {
	normalized := NormalizeQuery(term)
	if normalized == "" {
		return false
	}
	terms := s.terms.Load().(map[string]struct{})
	_, ok := terms[normalized]
	return ok
}

func (s *StopList) Add(term string) bool {
	normalized := NormalizeQuery(term)
	if normalized == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.terms.Load().(map[string]struct{})
	if _, ok := current[normalized]; ok {
		return false
	}

	next := cloneTerms(current)
	next[normalized] = struct{}{}
	s.terms.Store(next)
	return true
}

func (s *StopList) Delete(term string) bool {
	normalized := NormalizeQuery(term)
	if normalized == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.terms.Load().(map[string]struct{})
	if _, ok := current[normalized]; !ok {
		return false
	}

	next := cloneTerms(current)
	delete(next, normalized)
	s.terms.Store(next)
	return true
}

func (s *StopList) List() []string {
	current := s.terms.Load().(map[string]struct{})
	terms := make([]string, 0, len(current))
	for term := range current {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	return terms
}

func cloneTerms(src map[string]struct{}) map[string]struct{} {
	dst := make(map[string]struct{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
