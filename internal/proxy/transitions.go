package proxy

import (
	"sync"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
)

// transitionState tracks the first-request timestamp for each route that has
// transitions defined. The key is "<METHOD> <match>" e.g. "GET /orders/*".
type transitionState struct {
	mu       sync.Mutex
	firstHit map[string]time.Time
}

func newTransitionState() *transitionState {
	return &transitionState{
		firstHit: make(map[string]time.Time),
	}
}

// Reset clears all recorded first-hit times, restarting every transition
// sequence. Called on config hot-reload.
func (ts *transitionState) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.firstHit = make(map[string]time.Time)
}

// routeKey returns the unique key for a route.
func routeKey(route *config.Route) string {
	return route.Method + " " + route.Match
}

// resolve returns the case name that should be served for a route with
// transitions, recording the first-hit time if this is the first request.
//
// Resolution logic:
//  1. Record time.Now() as t0 on the very first request.
//  2. elapsed = time.Since(t0)
//  3. Walk transitions in order. Each entry with After > 0 is a threshold.
//     Serve the last entry whose threshold has NOT yet been crossed.
//     If all thresholds are crossed, serve the last (terminal) entry.
//
// Example with after=[30, 90, 0]:
//
//	elapsed < 30s  → transitions[0].Case
//	30s ≤ elapsed < 90s → transitions[1].Case
//	elapsed ≥ 90s  → transitions[2].Case  (terminal, After == 0)
func (ts *transitionState) resolve(route *config.Route) string {
	if len(route.Transitions) == 0 {
		return ""
	}

	key := routeKey(route)

	ts.mu.Lock()
	t0, seen := ts.firstHit[key]
	if !seen {
		t0 = time.Now()
		ts.firstHit[key] = t0
	}
	ts.mu.Unlock()

	elapsed := time.Since(t0)
	elapsedSecs := int(elapsed.Seconds())

	// Walk transitions: serve the last one whose After threshold hasn't been
	// reached yet. The terminal entry (After == 0) always matches.
	current := route.Transitions[len(route.Transitions)-1].Case

	for i := 0; i < len(route.Transitions)-1; i++ {
		t := route.Transitions[i]
		if t.After == 0 || elapsedSecs < t.After {
			current = t.Case
			break
		}
	}

	return current
}
