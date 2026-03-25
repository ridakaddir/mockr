package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/persist"
)

// transitionScheduler manages background goroutines that apply deferred file
// mutations after a resource is created via persist append.
//
// When a POST route has transitions defined, each non-terminal transition case
// with persist=true schedules a goroutine that sleeps for the cumulative
// duration, then applies the case's merge+defaults to the created file.
//
// This allows resources to "transition" on disk (e.g. Deploying → Ready)
// without requiring a subsequent GET request to trigger the state change.
type transitionScheduler struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// newTransitionScheduler creates a scheduler with a cancellable context.
// The parent context should be the server-level context so that shutdown
// cancels all pending mutations.
func newTransitionScheduler(parent context.Context) *transitionScheduler {
	ctx, cancel := context.WithCancel(parent)
	return &transitionScheduler{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Schedule inspects a route's transitions and spawns background goroutines
// for each transition case that needs a deferred file mutation.
//
// filePath is the absolute path to the file created by the append operation.
// configDir is needed to resolve relative defaults file paths.
//
// Only transition cases (beyond the initial/fallback case) that have
// persist=true and merge="update" are scheduled. The initial case is skipped
// because it was already applied by the append operation.
func (s *transitionScheduler) Schedule(route *config.Route, filePath, configDir string) {
	if len(route.Transitions) < 2 {
		return // need at least an initial + one transition case
	}

	// Accumulate durations of preceding entries to compute absolute delays
	// from now (creation time). The delay for entry N is the sum of durations
	// of all entries before it (entries 0..N-1).
	var cumulative int64
	for i := 0; i < len(route.Transitions); i++ {
		// Skip the first entry (it's the initial state, already applied by
		// the append operation).
		if i == 0 {
			cumulative += int64(route.Transitions[i].Duration)
			continue
		}

		t := route.Transitions[i]
		c, ok := route.Cases[t.Case]
		if !ok {
			cumulative += int64(t.Duration)
			continue
		}

		// Only schedule if the case has persist + defaults.
		if c.Persist && c.Defaults != "" {
			delay := time.Duration(cumulative) * time.Second
			s.schedule(delay, filePath, c, configDir)
		}

		// Accumulate this entry's duration for subsequent entries.
		cumulative += int64(t.Duration)
	}
}

// schedule spawns a single goroutine that waits for delay, then applies
// the case's defaults to the file via persist.Update.
func (s *transitionScheduler) schedule(delay time.Duration, filePath string, c config.Case, configDir string) {
	s.mu.Lock()
	ctx := s.ctx
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()

		select {
		case <-time.After(delay):
			// Load defaults and apply the mutation.
			incoming := loadDefaultsStatic(c.Defaults, configDir)
			if len(incoming) == 0 {
				logger.Warn("deferred transition: no defaults to apply",
					"file", filePath, "defaults", c.Defaults)
				return
			}

			if _, err := persist.Update(filePath, incoming); err != nil {
				if persist.IsNotFound(err) {
					logger.Warn("deferred transition: file was deleted before transition",
						"file", filePath)
				} else {
					logger.Error("deferred transition: update failed",
						"file", filePath, "err", err)
				}
				return
			}

			logger.Info("deferred transition applied",
				"file", filePath, "defaults", c.Defaults, "delay", delay.String())

		case <-ctx.Done():
			// Cancelled (hot-reload or shutdown).
			return
		}
	}()
}

// Reset cancels all pending mutations and creates a fresh context.
// Called on config hot-reload to restart transition schedules.
func (s *transitionScheduler) Reset(parent context.Context) {
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()

	// Wait for all goroutines to finish before creating the new context,
	// so there's no overlap between old and new goroutines.
	s.wg.Wait()

	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.ctx = ctx
	s.cancel = cancel
	s.mu.Unlock()
}

// Stop cancels all pending mutations and waits for goroutines to finish.
// Called on server shutdown.
func (s *transitionScheduler) Stop() {
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
}
