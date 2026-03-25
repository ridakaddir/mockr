package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/persist"
)

// applyPersist reads the stub file, mutates the array according to merge strategy,
// writes the file back, and sends the appropriate HTTP response.
//
// Returns (handled, createdFilePath). handled is true if the response was written
// (caller should not write anything else). createdFilePath is non-empty only for
// merge="append" operations, providing the path of the newly created file.
func applyPersist(w http.ResponseWriter, r *http.Request, c config.Case, bodyBytes []byte, routePattern string, configDir string, pathParams map[string]string) (bool, string) {
	if !c.Persist {
		return false, ""
	}

	filePath := resolveFilePath(c.File, r, bodyBytes, configDir, routePattern, pathParams)

	// Parse request body for incoming record.
	incoming := parseJSONBody(bodyBytes)

	switch strings.ToLower(c.Merge) {
	case "update":
		// Apply defaults if specified (enrich incoming data before persisting).
		incoming = loadDefaults(c.Defaults, incoming, r, bodyBytes, configDir, routePattern, pathParams)
		updated, err := persist.Update(filePath, incoming)
		if err != nil {
			if persist.IsNotFound(err) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			} else if persist.IsConfigError(err) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			} else {
				logger.Error("persist update", "file", filePath, "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub file error"})
			}
			return true, ""
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), updated)

	case "append":
		if !isDirectoryPath(filePath, c.File) {
			logger.Error("persist append", "file", filePath, "err", "append requires directory path")
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "merge=\"append\" requires file to point to a directory (path ending with /)",
			})
			return true, ""
		}
		// Apply defaults if specified (enrich incoming data before persisting).
		incoming = loadDefaults(c.Defaults, incoming, r, bodyBytes, configDir, routePattern, pathParams)

		// Key resolution: if the key field is configured but missing from the
		// incoming record (after defaults), resolve it from path params, then
		// wildcards, then query parameters. This allows named parameters like
		// {endpointId} in the route match pattern to be used as the filename
		// when the body doesn't contain the key field.
		if c.Key != "" {
			if v, exists := incoming[c.Key]; !exists || v == nil || fmt.Sprintf("%v", v) == "" {
				if resolved, ok := extractValue("path", c.Key, r, bodyBytes, routePattern, pathParams); ok {
					incoming[c.Key] = resolved
				} else if resolved, ok := extractValue("query", c.Key, r, bodyBytes, routePattern, pathParams); ok {
					incoming[c.Key] = resolved
				}
			}
		}

		createdPath, result, err := persist.AppendToDir(filePath, c.Key, incoming)
		if err != nil {
			logger.Error("persist append to dir", "dir", filePath, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub directory error"})
			return true, ""
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), result)
		return true, createdPath

	case "delete":
		if err := persist.DeleteFile(filePath); err != nil {
			if persist.IsNotFound(err) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			} else {
				logger.Error("persist delete file", "file", filePath, "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			return true, ""
		}
		logger.SetSource(w, logger.SourceStub)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(c.StatusCode())

	default:
		logger.Warn("persist: unknown merge strategy", "merge", c.Merge)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("unknown merge strategy %q. Valid options: update, append, delete", c.Merge),
		})
	}

	return true, ""
}

// resolveFilePath applies dynamic placeholders and configDir to a file path.
func resolveFilePath(filePath string, r *http.Request, bodyBytes []byte, configDir string, routePattern string, pathParams map[string]string) string {
	if hasDynamicPlaceholders(filePath) {
		filePath = resolveDynamicFile(filePath, r, bodyBytes, routePattern, pathParams)
	}
	return absPath(filePath, configDir)
}

// loadDefaults reads a defaults JSON file, resolves template tokens ({{uuid}},
// {{now}}, {{timestamp}}), and deep-merges the result under the incoming data
// so that incoming (request body) fields win on conflicts.
//
// Returns incoming unchanged if defaults is empty or on any error (warnings logged).
func loadDefaults(defaults string, incoming map[string]interface{},
	r *http.Request, bodyBytes []byte, configDir, routePattern string,
	pathParams map[string]string) map[string]interface{} {

	if defaults == "" {
		return incoming
	}

	defaultsPath := resolveFilePath(defaults, r, bodyBytes, configDir, routePattern, pathParams)

	// Ensure resolved path stays within configDir to prevent directory traversal.
	if configDir != "" {
		absConfig, _ := filepath.Abs(configDir)
		absDefaults, _ := filepath.Abs(defaultsPath)
		if !strings.HasPrefix(absDefaults, absConfig+string(filepath.Separator)) && absDefaults != absConfig {
			logger.Warn("persist defaults: path escapes config directory", "file", defaultsPath, "configDir", configDir)
			return incoming
		}
	}

	defaultsData, err := os.ReadFile(defaultsPath)
	if err != nil {
		logger.Warn("persist defaults: cannot read file", "file", defaultsPath, "err", err)
		return incoming
	}

	resolved, err := renderTemplate(string(defaultsData))
	if err != nil {
		logger.Warn("persist defaults: template error", "file", defaultsPath, "err", err)
		return incoming
	}

	var base map[string]interface{}
	if err := json.Unmarshal([]byte(resolved), &base); err != nil {
		logger.Warn("persist defaults: invalid JSON", "file", defaultsPath, "err", err)
		return incoming
	}

	return persist.DeepMerge(base, incoming)
}

// loadDefaultsStatic reads a defaults JSON file and returns its content as a map.
// Unlike loadDefaults, it does not require an HTTP request context — it only
// resolves the file path relative to configDir and runs template rendering.
//
// Used by the transition scheduler for deferred (background) file mutations
// where no request is available.
//
// Returns nil if defaults is empty, the file cannot be read, or parsing fails.
func loadDefaultsStatic(defaults, configDir string) map[string]interface{} {
	if defaults == "" {
		return nil
	}

	defaultsPath := absPath(defaults, configDir)

	// Ensure resolved path stays within configDir to prevent directory traversal.
	if configDir != "" {
		absConfig, _ := filepath.Abs(configDir)
		absDefaults, _ := filepath.Abs(defaultsPath)
		if !strings.HasPrefix(absDefaults, absConfig+string(filepath.Separator)) && absDefaults != absConfig {
			logger.Warn("deferred defaults: path escapes config directory", "file", defaultsPath, "configDir", configDir)
			return nil
		}
	}

	defaultsData, err := os.ReadFile(defaultsPath)
	if err != nil {
		logger.Warn("deferred defaults: cannot read file", "file", defaultsPath, "err", err)
		return nil
	}

	resolved, err := renderTemplate(string(defaultsData))
	if err != nil {
		logger.Warn("deferred defaults: template error", "file", defaultsPath, "err", err)
		return nil
	}

	var base map[string]interface{}
	if err := json.Unmarshal([]byte(resolved), &base); err != nil {
		logger.Warn("deferred defaults: invalid JSON", "file", defaultsPath, "err", err)
		return nil
	}

	return base
}

// isDirectoryPath determines if a file path should be treated as a directory
// for stub aggregation. Returns true if the path is an existing directory OR
// if the original config path ended with "/" (indicating directory intent).
func isDirectoryPath(resolvedPath, originalConfigFile string) bool {
	info, err := os.Stat(resolvedPath)
	if err == nil && info.IsDir() {
		return true
	}
	// If path doesn't exist but original config indicated directory intent
	if os.IsNotExist(err) && strings.HasSuffix(originalConfigFile, "/") {
		return true
	}
	return false
}
