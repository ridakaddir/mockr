package proxy

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/persist"
)

// applyPersist reads the stub file, mutates the array according to merge strategy,
// writes the file back, and sends the appropriate HTTP response.
//
// Returns true if it handled the response (caller should not write anything else).
func applyPersist(w http.ResponseWriter, r *http.Request, c config.Case, bodyBytes []byte, routePattern string, configDir string, pathParams map[string]string) bool {
	if !c.Persist {
		return false
	}

	filePath := resolveFilePath(c.File, r, bodyBytes, configDir, routePattern, pathParams)

	// Parse request body for incoming record.
	incoming := parseJSONBody(bodyBytes)

	switch strings.ToLower(c.Merge) {
	case "update":
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
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), updated)

	case "append":
		if !isDirectoryPath(filePath, c.File) {
			logger.Error("persist append", "file", filePath, "err", "append requires directory path")
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "merge=\"append\" requires file to point to a directory (path ending with /)",
			})
			return true
		}
		result, err := persist.AppendToDir(filePath, c.Key, incoming)
		if err != nil {
			logger.Error("persist append to dir", "dir", filePath, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stub directory error"})
			return true
		}
		logger.SetSource(w, logger.SourceStub)
		writeJSON(w, c.StatusCode(), result)

	case "delete":
		if err := persist.DeleteFile(filePath); err != nil {
			if persist.IsNotFound(err) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			} else {
				logger.Error("persist delete file", "file", filePath, "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			return true
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

	return true
}

// resolveFilePath applies dynamic placeholders and configDir to a file path.
func resolveFilePath(filePath string, r *http.Request, bodyBytes []byte, configDir string, routePattern string, pathParams map[string]string) string {
	if hasDynamicPlaceholders(filePath) {
		filePath = resolveDynamicFile(filePath, r, bodyBytes, routePattern, pathParams)
	}
	return absPath(filePath, configDir)
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
