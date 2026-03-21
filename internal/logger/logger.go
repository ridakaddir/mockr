package logger

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var l *log.Logger

func init() {
	l = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})
}

// Middleware wraps an http.Handler and logs each request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		latency := time.Since(start)
		logRequest(r.Method, r.URL.RequestURI(), rw.status, latency, rw.source)
	})
}

// logRequest prints a single request line.
func logRequest(method, path string, status int, latency time.Duration, source string) {
	src := source
	if src == "" {
		src = "unknown"
	}
	msg := fmt.Sprintf("%s %s", method, path)

	switch {
	case status >= 500:
		l.Error(msg, "status", status, "latency", latency, "via", src)
	case status >= 400:
		l.Warn(msg, "status", status, "latency", latency, "via", src)
	default:
		l.Info(msg, "status", status, "latency", latency, "via", src)
	}
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	l.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	l.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	l.Error(msg, args...)
}

// Fatal logs a fatal message and exits.
func Fatal(msg string, args ...any) {
	l.Fatal(msg, args...)
}

// Source values used by SetSource.
const (
	SourceStub  = "stub"  // response served from a local mock/stub file
	SourceProxy = "proxy" // response forwarded to upstream
)

// responseWriter wraps http.ResponseWriter to capture the status code and source.
type responseWriter struct {
	http.ResponseWriter
	status int
	source string
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// SetSource walks the ResponseWriter chain to find and set the source label
// on the logger's responseWriter. This handles cases where w has been wrapped
// by an intermediate writer (e.g. capturingWriter in record mode).
func SetSource(w http.ResponseWriter, source string) {
	type unwrapper interface {
		Unwrap() http.ResponseWriter
	}

	current := w
	for current != nil {
		if rw, ok := current.(*responseWriter); ok {
			rw.source = source
			return
		}
		if u, ok := current.(unwrapper); ok {
			current = u.Unwrap()
		} else {
			return
		}
	}
}
