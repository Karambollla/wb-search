package core

import (
	"sync"
	"time"
)

type bucket struct {
	index  int64
	counts map[string]int64
}

type BucketedCounter struct {
	mu             sync.RWMutex
	buckets        []bucket
	bucketDuration time.Duration
}

func NewBucketedCounter(window, bucketDuration time.Duration) *BucketedCounter {
	if bucketDuration <= 0 {
		bucketDuration = 5 * time.Second
	}
	bucketCount := int(window / bucketDuration)
	if bucketCount < 1 {
		bucketCount = 1
	}

	buckets := make([]bucket, bucketCount)
	for i := range buckets {
		buckets[i] = bucket{index: -1, counts: make(map[string]int64)}
	}

	return &BucketedCounter{
		buckets:        buckets,
		bucketDuration: bucketDuration,
	}
}

func (c *BucketedCounter) Add(query string, at time.Time) {
	if query == "" {
		return
	}

	index := c.bucketIndex(at)
	slot := int(index % int64(len(c.buckets)))

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.buckets[slot].index > index {
		return
	}
	if c.buckets[slot].index != index {
		c.buckets[slot].index = index
		c.buckets[slot].counts = make(map[string]int64)
	}
	c.buckets[slot].counts[query]++
}

func (c *BucketedCounter) Snapshot(now time.Time) map[string]int64 {
	current := c.bucketIndex(now)
	result := make(map[string]int64)

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, b := range c.buckets {
		if b.index < 0 {
			continue
		}
		age := current - b.index
		if age < 0 || age >= int64(len(c.buckets)) {
			continue
		}
		for query, count := range b.counts {
			result[query] += count
		}
	}

	return result
}

func (c *BucketedCounter) bucketIndex(t time.Time) int64 {
	return t.UnixNano() / c.bucketDuration.Nanoseconds()
}
