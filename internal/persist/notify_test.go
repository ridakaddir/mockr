package persist

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotify_ListenerReceivesEvents(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	var received []ChangeEvent
	var mu sync.Mutex

	OnChange(func(events []ChangeEvent) {
		mu.Lock()
		received = append(received, events...)
		mu.Unlock()
	})

	notify("/tmp/stubs/test.json", FileCreated)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1)
	assert.Equal(t, "/tmp/stubs/test.json", received[0].FilePath)
	assert.Equal(t, "/tmp/stubs", received[0].Dir)
	assert.Equal(t, FileCreated, received[0].Type)
}

func TestNotify_MultipleListeners(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	var count1, count2 int
	var mu sync.Mutex

	OnChange(func(events []ChangeEvent) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	OnChange(func(events []ChangeEvent) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	notify("/tmp/file.json", FileUpdated)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, count1)
	assert.Equal(t, 1, count2)
}

func TestNotify_NoListeners(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	// Should not panic with no listeners.
	notify("/tmp/file.json", FileDeleted)
}

func TestNotify_EventDir(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	var received []ChangeEvent
	OnChange(func(events []ChangeEvent) {
		received = append(received, events...)
	})

	notify("/config/stubs/deployments/ep-123/dep-456.json", FileCreated)

	require.Len(t, received, 1)
	assert.Equal(t, "/config/stubs/deployments/ep-123", received[0].Dir)
}

func TestResetListeners(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	called := false
	OnChange(func(events []ChangeEvent) {
		called = true
	})

	ResetListeners()
	notify("/tmp/file.json", FileCreated)

	assert.False(t, called, "listener should not be called after reset")
}

func TestOnChange_Unsubscribe(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	var count int
	unsub := OnChange(func(events []ChangeEvent) {
		count++
	})

	notify("/tmp/file.json", FileCreated)
	assert.Equal(t, 1, count, "listener should be called before unsubscribe")

	unsub()

	notify("/tmp/file.json", FileUpdated)
	assert.Equal(t, 1, count, "listener should not be called after unsubscribe")
}

func TestNotify_PanicRecovery(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	var safeListenerCalled bool

	// Register a listener that panics.
	OnChange(func(events []ChangeEvent) {
		panic("test panic")
	})

	// Register a second listener that should still be called.
	OnChange(func(events []ChangeEvent) {
		safeListenerCalled = true
	})

	// Should not panic.
	notify("/tmp/file.json", FileCreated)
	assert.True(t, safeListenerCalled, "second listener should be called even when first panics")
}

// TestPersistOperationsNotify verifies that the actual persist operations
// (AppendToDir, Update, DeleteFile) trigger change notifications.
func TestPersistOperationsNotify(t *testing.T) {
	ResetListeners()
	defer ResetListeners()

	dir := t.TempDir()

	var events []ChangeEvent
	var mu sync.Mutex
	OnChange(func(evts []ChangeEvent) {
		mu.Lock()
		events = append(events, evts...)
		mu.Unlock()
	})

	// --- AppendToDir triggers FileCreated ---
	createdPath, _, err := AppendToDir(dir, "id", map[string]interface{}{
		"id":   "item-1",
		"name": "test",
	})
	require.NoError(t, err)

	// Wait briefly for async delivery.
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Len(t, events, 1, "AppendToDir should trigger one event")
	assert.Equal(t, createdPath, events[0].FilePath)
	assert.Equal(t, dir, events[0].Dir)
	assert.Equal(t, FileCreated, events[0].Type)
	events = nil
	mu.Unlock()

	// --- Update triggers FileUpdated ---
	_, err = Update(createdPath, map[string]interface{}{"status": "ready"})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Len(t, events, 1, "Update should trigger one event")
	assert.Equal(t, createdPath, events[0].FilePath)
	assert.Equal(t, FileUpdated, events[0].Type)
	events = nil
	mu.Unlock()

	// --- DeleteFile triggers FileDeleted ---
	err = DeleteFile(createdPath)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Len(t, events, 1, "DeleteFile should trigger one event")
	assert.Equal(t, createdPath, events[0].FilePath)
	assert.Equal(t, filepath.Dir(createdPath), events[0].Dir)
	assert.Equal(t, FileDeleted, events[0].Type)
	mu.Unlock()
}
