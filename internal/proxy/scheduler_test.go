package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubConfigLoader is a minimal configLoader implementation for handler tests.
type stubConfigLoader struct {
	cfg       *config.Config
	configDir string
}

func (s *stubConfigLoader) Get() *config.Config         { return s.cfg }
func (s *stubConfigLoader) AddRoute(route config.Route) {}
func (s *stubConfigLoader) ConfigDir() string           { return s.configDir }

// writeJSONFile writes a map to a JSON file.
func writeJSONFile(t *testing.T, path string, data map[string]interface{}) {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0644))
}

func TestScheduler_FiresAfterDuration(t *testing.T) {
	dir := t.TempDir()

	// Create the stub file (simulating what AppendToDir would create).
	stubFile := filepath.Join(dir, "dep-123.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"id":     "dep-123",
		"status": "Deploying",
	})

	// Create the defaults file.
	defaultsFile := filepath.Join(dir, "ready-defaults.json")
	require.NoError(t, os.WriteFile(defaultsFile, []byte(`{"status": "Ready"}`), 0644))

	route := &config.Route{
		Method:   "POST",
		Match:    "/deployments",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "deploying", Duration: 1}, // 1 second for fast test
			{Case: "ready"},
		},
		Cases: map[string]config.Case{
			"created": {
				Status:  201,
				Persist: true,
				Merge:   "append",
			},
			"ready": {
				Persist:  true,
				Merge:    "update",
				Defaults: "ready-defaults.json",
			},
		},
	}

	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	sched.Schedule(route, stubFile, dir)

	// Verify file is still "Deploying" immediately.
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "Deploying", data["status"], "should still be Deploying immediately")

	// Wait for the transition to fire (1s duration + buffer).
	time.Sleep(1500 * time.Millisecond)

	// Verify file has transitioned to "Ready".
	data = readJSONFile(t, stubFile)
	assert.Equal(t, "Ready", data["status"], "should have transitioned to Ready")
	assert.Equal(t, "dep-123", data["id"], "original fields should be preserved")
}

func TestScheduler_CancelledOnStop(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "resource.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"status": "initial",
	})

	defaultsFile := filepath.Join(dir, "update.json")
	require.NoError(t, os.WriteFile(defaultsFile, []byte(`{"status": "updated"}`), 0644))

	route := &config.Route{
		Method:   "POST",
		Match:    "/resources",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "initial", Duration: 5}, // long enough that Stop cancels first
			{Case: "updated"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"updated": {Persist: true, Merge: "update", Defaults: "update.json"},
		},
	}

	sched := newTransitionScheduler(context.Background())
	sched.Schedule(route, stubFile, dir)

	// Stop immediately — should cancel the pending mutation.
	sched.Stop()

	// Verify file was NOT updated.
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "initial", data["status"], "mutation should have been cancelled")
}

func TestScheduler_CancelledOnReset(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "resource.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"status": "initial",
	})

	defaultsFile := filepath.Join(dir, "update.json")
	require.NoError(t, os.WriteFile(defaultsFile, []byte(`{"status": "updated"}`), 0644))

	route := &config.Route{
		Method:   "POST",
		Match:    "/resources",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "initial", Duration: 5},
			{Case: "updated"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"updated": {Persist: true, Merge: "update", Defaults: "update.json"},
		},
	}

	sched := newTransitionScheduler(context.Background())
	sched.Schedule(route, stubFile, dir)

	// Reset (simulating hot-reload) — should cancel the pending mutation.
	sched.Reset(context.Background())

	// Verify file was NOT updated.
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "initial", data["status"], "mutation should have been cancelled on reset")

	// Scheduler should still be usable after reset.
	sched.Stop()
}

func TestScheduler_MultipleTransitions(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "item.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"id":    "item-1",
		"phase": "building",
	})

	phase2File := filepath.Join(dir, "phase2.json")
	require.NoError(t, os.WriteFile(phase2File, []byte(`{"phase": "testing"}`), 0644))

	phase3File := filepath.Join(dir, "phase3.json")
	require.NoError(t, os.WriteFile(phase3File, []byte(`{"phase": "done"}`), 0644))

	// Three states: building (1s) → testing (1s) → done (terminal)
	route := &config.Route{
		Method:   "POST",
		Match:    "/items",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "building", Duration: 1},
			{Case: "testing", Duration: 1},
			{Case: "done"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"testing": {Persist: true, Merge: "update", Defaults: "phase2.json"},
			"done":    {Persist: true, Merge: "update", Defaults: "phase3.json"},
		},
	}

	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	sched.Schedule(route, stubFile, dir)

	// At t=0, should be "building".
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "building", data["phase"])

	// After 1.5s, should be "testing" (first transition at 1s).
	time.Sleep(1500 * time.Millisecond)
	data = readJSONFile(t, stubFile)
	assert.Equal(t, "testing", data["phase"], "should have transitioned to testing at t=1s")

	// After another 1s (total ~2.5s), should be "done" (second transition at 2s).
	time.Sleep(1 * time.Second)
	data = readJSONFile(t, stubFile)
	assert.Equal(t, "done", data["phase"], "should have transitioned to done at t=2s")
	assert.Equal(t, "item-1", data["id"], "original fields should be preserved")
}

func TestScheduler_NoPersistCaseSkipped(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "resource.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"status": "initial",
	})

	defaultsFile := filepath.Join(dir, "update.json")
	require.NoError(t, os.WriteFile(defaultsFile, []byte(`{"status": "should-not-appear"}`), 0644))

	// The "updated" case does NOT have persist=true — it should be skipped.
	route := &config.Route{
		Method:   "POST",
		Match:    "/resources",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "initial", Duration: 1},
			{Case: "updated"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"updated": {Persist: false, Merge: "update", Defaults: "update.json"},
		},
	}

	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	sched.Schedule(route, stubFile, dir)

	// Wait enough time for it to have fired if it were scheduled.
	time.Sleep(1500 * time.Millisecond)

	// Verify file was NOT updated (the case had persist=false).
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "initial", data["status"], "non-persist case should not trigger mutation")
}

func TestScheduler_NoDefaultsSkipped(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "resource.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"status": "initial",
	})

	// The "updated" case has persist=true but NO defaults — nothing to merge.
	route := &config.Route{
		Method:   "POST",
		Match:    "/resources",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "initial", Duration: 1},
			{Case: "updated"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"updated": {Persist: true, Merge: "update", Defaults: ""},
		},
	}

	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	sched.Schedule(route, stubFile, dir)

	// Wait enough time.
	time.Sleep(1500 * time.Millisecond)

	// File should be unchanged — no defaults means no mutation.
	data := readJSONFile(t, stubFile)
	assert.Equal(t, "initial", data["status"], "no-defaults case should not trigger mutation")
}

func TestScheduler_DeletedFileBeforeTransition(t *testing.T) {
	dir := t.TempDir()

	stubFile := filepath.Join(dir, "resource.json")
	writeJSONFile(t, stubFile, map[string]interface{}{
		"status": "initial",
	})

	defaultsFile := filepath.Join(dir, "update.json")
	require.NoError(t, os.WriteFile(defaultsFile, []byte(`{"status": "updated"}`), 0644))

	route := &config.Route{
		Method:   "POST",
		Match:    "/resources",
		Fallback: "created",
		Transitions: []config.Transition{
			{Case: "initial", Duration: 1},
			{Case: "updated"},
		},
		Cases: map[string]config.Case{
			"created": {Persist: true, Merge: "append"},
			"updated": {Persist: true, Merge: "update", Defaults: "update.json"},
		},
	}

	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	sched.Schedule(route, stubFile, dir)

	// Delete the file before the transition fires.
	require.NoError(t, os.Remove(stubFile))

	// Wait for the transition to fire.
	time.Sleep(1500 * time.Millisecond)

	// The file should still not exist (the scheduler should log a warning, not crash).
	assert.NoFileExists(t, stubFile, "deleted file should not be recreated")
}

// ---------------------------------------------------------------------------
// Full integration: POST → background transition → GET reads updated file
// ---------------------------------------------------------------------------

// TestPostCreatesAndTransitionsInBackground exercises the full Handler.ServeHTTP
// flow with the deployment scenario:
//
//  1. POST creates a deployment with "status": "Deploying"
//  2. The scheduler fires a background mutation after 1s
//  3. GET by ID returns "status": "Ready"
//  4. GET list (directory aggregation) also returns "Ready"
func TestPostCreatesAndTransitionsInBackground(t *testing.T) {
	dir := t.TempDir()

	// Create directories.
	deploymentsDir := filepath.Join(dir, "deployments", "ep-123")
	require.NoError(t, os.MkdirAll(deploymentsDir, 0755))

	defaultsDir := filepath.Join(dir, "defaults")
	require.NoError(t, os.MkdirAll(defaultsDir, 0755))

	// Defaults: initial deployment template.
	require.NoError(t, os.WriteFile(
		filepath.Join(defaultsDir, "deployment.json"),
		[]byte(`{"status": "Deploying", "region": "us-east-1"}`),
		0644,
	))

	// Defaults: ready transition template.
	require.NoError(t, os.WriteFile(
		filepath.Join(defaultsDir, "deployment-ready.json"),
		[]byte(`{"status": "Ready"}`),
		0644,
	))

	// --- Config ---

	cfg := &config.Config{
		Routes: []config.Route{
			// POST — creates deployment, transitions defined here
			{
				Method:   "POST",
				Match:    "/api/v1/*/environments/*/endpoint/{endpointId}/deployment",
				Fallback: "created",
				Transitions: []config.Transition{
					{Case: "deploying", Duration: 1}, // 1s for fast test
					{Case: "ready"},
				},
				Cases: map[string]config.Case{
					"created": {
						Status:   201,
						File:     "deployments/{path.endpointId}/",
						Persist:  true,
						Merge:    "append",
						Key:      "deploymentId",
						Defaults: "defaults/deployment.json",
					},
					"ready": {
						Persist:  true,
						Merge:    "update",
						Defaults: "defaults/deployment-ready.json",
					},
				},
			},
			// GET by ID — pure read
			{
				Method:   "GET",
				Match:    "/api/v1/*/environments/*/endpoint/{endpointId}/deployment/{deploymentId}",
				Fallback: "success",
				Cases: map[string]config.Case{
					"success": {
						Status: 200,
						File:   "deployments/{path.endpointId}/{path.deploymentId}.json",
					},
				},
			},
			// GET list — directory aggregation
			{
				Method:   "GET",
				Match:    "/api/v1/*/environments/*/endpoint/{endpointId}/deployment",
				Fallback: "success",
				Cases: map[string]config.Case{
					"success": {
						Status: 200,
						File:   "deployments/{path.endpointId}/",
					},
				},
			},
		},
	}

	loader := &stubConfigLoader{cfg: cfg, configDir: dir}
	sched := newTransitionScheduler(context.Background())
	defer sched.Stop()

	handler := NewHandlerWithTransitions(loader, nil, false, "", newTransitionState(), sched)

	// --- Step 1: POST to create the deployment ---

	postBody := bytes.NewBufferString(`{"deploymentId": "dep-456", "name": "my-deployment"}`)
	postReq := httptest.NewRequest(http.MethodPost,
		"/api/v1/org1/environments/staging/endpoint/ep-123/deployment",
		postBody,
	)
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	handler.ServeHTTP(postW, postReq)

	require.Equal(t, http.StatusCreated, postW.Code, "POST should return 201")

	var postResp map[string]interface{}
	require.NoError(t, json.Unmarshal(postW.Body.Bytes(), &postResp))
	assert.Equal(t, "Deploying", postResp["status"], "initial status should be Deploying (from defaults)")
	assert.Equal(t, "dep-456", postResp["deploymentId"])
	assert.Equal(t, "my-deployment", postResp["name"])
	assert.Equal(t, "us-east-1", postResp["region"], "region from defaults should be present")

	// Verify the file was created on disk.
	createdFile := filepath.Join(deploymentsDir, "dep-456.json")
	require.FileExists(t, createdFile)
	diskData := readJSONFile(t, createdFile)
	assert.Equal(t, "Deploying", diskData["status"])

	// --- Step 2: GET by ID immediately → should see "Deploying" ---

	getReq := httptest.NewRequest(http.MethodGet,
		"/api/v1/org1/environments/staging/endpoint/ep-123/deployment/dep-456",
		nil,
	)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)

	require.Equal(t, http.StatusOK, getW.Code)
	var getResp map[string]interface{}
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &getResp))
	assert.Equal(t, "Deploying", getResp["status"],
		"GET immediately after POST should return Deploying")

	// --- Step 3: Wait for the background transition to fire ---

	time.Sleep(1500 * time.Millisecond)

	// --- Step 4: GET by ID → should now see "Ready" ---

	getReq2 := httptest.NewRequest(http.MethodGet,
		"/api/v1/org1/environments/staging/endpoint/ep-123/deployment/dep-456",
		nil,
	)
	getW2 := httptest.NewRecorder()
	handler.ServeHTTP(getW2, getReq2)

	require.Equal(t, http.StatusOK, getW2.Code)
	var getResp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(getW2.Body.Bytes(), &getResp2))
	assert.Equal(t, "Ready", getResp2["status"],
		"GET after background transition should return Ready")
	assert.Equal(t, "dep-456", getResp2["deploymentId"],
		"original fields should be preserved")
	assert.Equal(t, "my-deployment", getResp2["name"],
		"original fields should be preserved")

	// --- Step 5: Verify the file on disk was updated ---

	diskData2 := readJSONFile(t, createdFile)
	assert.Equal(t, "Ready", diskData2["status"],
		"file on disk should reflect the background transition")

	// --- Step 6: GET list → should include the updated deployment ---

	listReq := httptest.NewRequest(http.MethodGet,
		"/api/v1/org1/environments/staging/endpoint/ep-123/deployment",
		nil,
	)
	listW := httptest.NewRecorder()
	handler.ServeHTTP(listW, listReq)

	require.Equal(t, http.StatusOK, listW.Code)
	var listResp []map[string]interface{}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
	require.Len(t, listResp, 1, "should have one deployment")
	assert.Equal(t, "Ready", listResp[0]["status"],
		"list endpoint should also reflect the background transition")
}
