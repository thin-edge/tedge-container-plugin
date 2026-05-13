package app

import (
	"sync"
	"time"
)

// RestartTracker maintains a sliding-window history of container restart
// events. It is used to detect crash loops — containers that exit and restart
// faster than they can become healthy.
type RestartTracker struct {
	mu        sync.Mutex
	history   map[string][]time.Time
	window    time.Duration
	threshold int
}

// NewRestartTracker returns a RestartTracker that considers a container to be
// in a crash loop once it has accumulated at least threshold restart events
// within window.
func NewRestartTracker(window time.Duration, threshold int) *RestartTracker {
	return &RestartTracker{
		history:   make(map[string][]time.Time),
		window:    window,
		threshold: threshold,
	}
}

// Record records one restart event for name, evicts events outside the
// sliding window, and returns the current restart count together with whether
// the crash-loop threshold has been exceeded.
func (rt *RestartTracker) Record(name string) (count int, exceeded bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rt.window)

	prev := rt.history[name]
	// Re-use the backing array – evict events outside the window in-place.
	valid := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	valid = append(valid, now)
	rt.history[name] = valid

	count = len(valid)
	exceeded = count >= rt.threshold
	return count, exceeded
}

// Clear resets the restart history for name. Call this when a container
// becomes healthy or is permanently removed so that a subsequent restart
// storm is not counted against the earlier window.
func (rt *RestartTracker) Clear(name string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.history, name)
}
