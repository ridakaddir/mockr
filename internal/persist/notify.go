// Package persist — file mutation notification system.
//
// When a stub file is created, updated, or deleted, the package can notify
// registered listeners so that cross-references (e.g. endpoint → deployments
// directory) can be re-evaluated automatically.
package persist

import (
	"path/filepath"
	"sync"

	"github.com/ridakaddir/mockr/internal/logger"
)

// ChangeType describes what happened to a stub file.
type ChangeType int

const (
	FileCreated ChangeType = iota
	FileUpdated
	FileDeleted
)

// ChangeEvent describes a single stub file mutation.
type ChangeEvent struct {
	FilePath string     // Absolute path to the file that changed.
	Dir      string     // Parent directory of the changed file.
	Type     ChangeType // What happened.
}

// ChangeListener is called when one or more stub files have been mutated.
// Events are delivered in a batch to allow efficient bulk processing.
type ChangeListener func(events []ChangeEvent)

// listenerEntry wraps a listener with an active flag for safe removal.
type listenerEntry struct {
	fn     ChangeListener
	active bool
}

var (
	listenersMu sync.RWMutex
	listeners   []*listenerEntry
)

// OnChange registers a callback invoked whenever stub files are mutated via
// AppendToDir, Update, or DeleteFile. Returns an unsubscribe function that
// removes the listener. Safe for concurrent use.
func OnChange(fn ChangeListener) func() {
	listenersMu.Lock()
	entry := &listenerEntry{fn: fn, active: true}
	listeners = append(listeners, entry)
	listenersMu.Unlock()

	return func() {
		listenersMu.Lock()
		entry.active = false
		listenersMu.Unlock()
	}
}

// ResetListeners removes all registered change listeners.
// Intended for tests that need a clean state.
func ResetListeners() {
	listenersMu.Lock()
	defer listenersMu.Unlock()
	listeners = nil
}

// notify delivers a single change event to all registered listeners.
// Each listener is called inside a recover block to prevent a panicking
// listener from crashing the server.
func notify(filePath string, changeType ChangeType) {
	listenersMu.RLock()
	entries := make([]*listenerEntry, len(listeners))
	copy(entries, listeners)
	listenersMu.RUnlock()

	if len(entries) == 0 {
		return
	}

	event := ChangeEvent{
		FilePath: filePath,
		Dir:      filepath.Dir(filePath),
		Type:     changeType,
	}
	batch := []ChangeEvent{event}

	for _, entry := range entries {
		if !entry.active {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("persist change listener panicked", "err", r)
				}
			}()
			entry.fn(batch)
		}()
	}
}
