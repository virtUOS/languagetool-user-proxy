package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const cleanupInterval = 5 * time.Minute
const entryTTL = 10 * time.Minute

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Store holds a per-key token bucket limiter.
type Store struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rps      rate.Limit
	burst    int
}

func NewStore(rps float64, burst int) *Store {
	s := &Store{
		limiters: make(map[string]*entry),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go s.cleanup()
	return s
}

// Allow returns true if the key is within its rate limit.
func (s *Store) Allow(key string) bool {
	s.mu.Lock()
	e, ok := s.limiters[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(s.rps, s.burst)}
		s.limiters[key] = e
	}
	e.lastSeen = time.Now()
	s.mu.Unlock()
	return e.limiter.Allow()
}

func (s *Store) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-entryTTL)
		s.mu.Lock()
		for key, e := range s.limiters {
			if e.lastSeen.Before(cutoff) {
				delete(s.limiters, key)
			}
		}
		s.mu.Unlock()
	}
}
