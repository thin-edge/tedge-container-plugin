package app

import (
	"sync"
	"time"
)

// EventRateLimiter enforces a minimum interval between events per key.
// It is used to prevent crash-looping containers from flooding the MQTT
// broker with engine event messages (start/die bursts at millisecond
// intervals can saturate the broker and starve other MQTT operations).
//
// The key is typically "<containerName>/<eventType>" so that each
// (container, event-type) pair is rate-limited independently.
type EventRateLimiter struct {
	mu       sync.Mutex
	last     map[string]time.Time
	interval time.Duration
}

// NewEventRateLimiter creates a limiter that passes at most one event per
// key per interval duration.
func NewEventRateLimiter(interval time.Duration) *EventRateLimiter {
	return &EventRateLimiter{
		last:     make(map[string]time.Time),
		interval: interval,
	}
}

// Allow returns true if at least interval has elapsed since the last allowed
// event for key, and records the current time as its new last-seen time.
// Returns false when the event should be suppressed.
func (r *EventRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if t, ok := r.last[key]; ok && now.Sub(t) < r.interval {
		return false
	}
	r.last[key] = now
	return true
}

// Remove clears the rate-limit history for key, effectively resetting its
// window. Call this when a container is removed or becomes healthy so that
// fresh events are not incorrectly suppressed.
func (r *EventRateLimiter) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.last, key)
}
