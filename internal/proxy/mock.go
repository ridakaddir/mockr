package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
)

// serveMock writes a mock response for the given case to w.
// bodyBytes is the buffered request body (needed for dynamic file resolution).
// configDir is the directory of the config file, used to resolve relative file paths.
func serveMock(w http.ResponseWriter, r *http.Request, c config.Case, bodyBytes []byte, configDir string) {
	if c.Delay > 0 {
		time.Sleep(time.Duration(c.Delay) * time.Second)
	}

	// Determine response body.
	var body []byte
	var err error

	switch {
	case c.File != "":
		filePath := c.File
		if hasDynamicPlaceholders(filePath) {
			filePath = resolveDynamicFile(filePath, r, bodyBytes)
		}
		// Resolve relative paths against the config file's directory.
		if configDir != "" && !filepath.IsAbs(filePath) {
			filePath = filepath.Join(configDir, filePath)
		}

		body, err = os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				// Signal caller that the file was not found so it can fall through.
				w.Header().Set("X-Mockr-File-Missing", "true")
				return
			}
			logger.Error("reading stub file", "file", filePath, "err", err)
			http.Error(w, `{"error":"stub file read error"}`, http.StatusInternalServerError)
			return
		}

	case c.JSON != "":
		rendered, err := renderTemplate(c.JSON)
		if err != nil {
			logger.Error("rendering json template", "err", err)
			http.Error(w, `{"error":"template render error"}`, http.StatusInternalServerError)
			return
		}
		body = []byte(rendered)

	default:
		body = []byte("{}")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c.StatusCode())
	_, _ = w.Write(body)
}

// renderTemplate processes {{uuid}}, {{now}}, {{timestamp}} tokens in a JSON string.
func renderTemplate(s string) (string, error) {
	funcMap := template.FuncMap{
		"uuid": func() string {
			return uuid.New().String()
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		"timestamp": func() string {
			return fmt.Sprintf("%d", time.Now().UnixMilli())
		},
	}

	// Convert {{token}} to Go template syntax {{call token}}.
	// Our tokens are zero-arg functions so we can call them directly.
	tmpl, err := template.New("mock").Funcs(funcMap).Parse(s)
	if err != nil {
		return s, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return s, err
	}

	return buf.String(), nil
}

// writeJSON writes a JSON-encoded value with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// isFileMissing checks whether serveMock signalled a missing dynamic file.
func isFileMissing(w http.ResponseWriter) bool {
	return w.Header().Get("X-Mockr-File-Missing") == "true"
}

// clearFileMissing removes the internal signal header before sending to client.
func clearFileMissing(w http.ResponseWriter) {
	w.Header().Del("X-Mockr-File-Missing")
}

// indentJSON pretty-prints raw JSON bytes.
func indentJSON(raw []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return raw
	}
	return buf.Bytes()
}

// slugify converts a URL path to a safe filename segment.
func slugify(s string) string {
	s = strings.TrimPrefix(s, "/")
	s = strings.ReplaceAll(s, "/", "_")
	r := strings.NewReplacer(
		"?", "_", "&", "_", "=", "_", " ", "_",
	)
	return r.Replace(s)
}
