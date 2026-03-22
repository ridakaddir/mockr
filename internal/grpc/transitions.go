package grpc

import (
	"sync"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
)

// grpcTransitionState tracks the first-request timestamp for each gRPC route
// that has transitions defined. Mirrors proxy.transitionState for GRPCRoute.
type grpcTransitionState struct {
	mu       sync.Mutex
	firstHit map[string]time.Time
}

func newGRPCTransitionState() *grpcTransitionState {
	return &grpcTransitionState{
		firstHit: make(map[string]time.Time),
	}
}

// Reset clears all recorded first-hit times (called on config hot-reload).
func (ts *grpcTransitionState) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.firstHit = make(map[string]time.Time)
}

// resolve returns the active case name for a gRPC route with transitions.
func (ts *grpcTransitionState) resolve(route *config.GRPCRoute) string {
	if len(route.Transitions) == 0 {
		return ""
	}

	key := route.Match
	ts.mu.Lock()
	t0, seen := ts.firstHit[key]
	if !seen {
		t0 = time.Now()
		ts.firstHit[key] = t0
	}
	ts.mu.Unlock()

	elapsed := time.Since(t0)
	elapsedSecs := int(elapsed.Seconds())

	// Default to terminal (last) entry.
	current := route.Transitions[len(route.Transitions)-1].Case

	for i := 0; i < len(route.Transitions)-1; i++ {
		t := route.Transitions[i]
		if t.After > 0 && elapsedSecs < t.After {
			current = t.Case
			break
		}
	}

	return current
}
