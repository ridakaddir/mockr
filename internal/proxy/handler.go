package proxy

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
)

// Handler holds runtime state for the request dispatcher.
type Handler struct {
	loader      configLoader
	rp          *httputil.ReverseProxy // nil if no --target
	rec         responseRecorder       // non-nil in record mode, created once
	apiPrefix   string                 // stripped from request path before matching + forwarding
	transitions *transitionState       // time-based transition state
}

// configLoader abstracts config.Loader so it can be mocked in tests.
type configLoader interface {
	Get() *config.Config
	AddRoute(route config.Route)
	ConfigDir() string
}

// NewHandler builds a new Handler with a fresh transition state.
func NewHandler(loader configLoader, rp *httputil.ReverseProxy, recordMode bool, apiPrefix string) *Handler {
	return NewHandlerWithTransitions(loader, rp, recordMode, apiPrefix, newTransitionState())
}

// NewHandlerWithTransitions builds a Handler with the provided transition state.
// Used by NewServer to share the transition state with the config reload callback.
func NewHandlerWithTransitions(loader configLoader, rp *httputil.ReverseProxy, recordMode bool, apiPrefix string, ts *transitionState) *Handler {
	// Normalise prefix: ensure it starts with / and has no trailing slash.
	if apiPrefix != "" && !strings.HasPrefix(apiPrefix, "/") {
		apiPrefix = "/" + apiPrefix
	}
	apiPrefix = strings.TrimRight(apiPrefix, "/")

	// Create the recorder once so its internal `seen` dedup map persists
	// across all requests for the lifetime of the server.
	var rec responseRecorder
	if recordMode {
		rec = recorder(loader.ConfigDir(), loader)
	}

	return &Handler{
		loader:      loader,
		rp:          rp,
		rec:         rec,
		apiPrefix:   apiPrefix,
		transitions: ts,
	}
}

// ServeHTTP is the main dispatcher.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg := h.loader.Get()
	bodyBytes := readBody(r)

	// Strip the api prefix from the path used for matching and forwarding.
	// The original r.URL.Path is preserved; matchPath and upstream see the stripped path.
	matchPath_ := r.URL.Path
	if h.apiPrefix != "" {
		if strings.HasPrefix(r.URL.Path, h.apiPrefix) {
			matchPath_ = r.URL.Path[len(h.apiPrefix):]
			if matchPath_ == "" {
				matchPath_ = "/"
			}
		}
	}

	for i := range cfg.Routes {
		route := &cfg.Routes[i]

		if !route.IsEnabled() {
			continue
		}
		if !strings.EqualFold(route.Method, r.Method) {
			continue
		}
		if !matchPath(route.Match, matchPath_) {
			continue
		}

		// Route matched — resolve case.
		caseName := h.resolveCase(route, r, bodyBytes)
		if caseName == "" {
			// No condition matched and no fallback — fall through to proxy.
			break
		}

		c, ok := route.Cases[caseName]
		if !ok {
			logger.Warn("case not found in route", "case", caseName, "route", route.Match)
			break
		}

		// Persist (mutating methods).
		if c.Persist {
			if handled := applyPersist(w, r, c, bodyBytes, route.Match, h.loader.ConfigDir()); handled {
				return
			}
		}

		// Serve mock response into a buffer so we can inspect it first.
		rec := newResponseRecorder(w)
		serveMock(rec, r, c, bodyBytes, h.loader.ConfigDir())

		// If the dynamic file was missing, try next condition / fallback / proxy.
		if isFileMissing(rec) {
			continue
		}

		// Mark the response as coming from a local stub before flushing.
		logger.SetSource(w, logger.SourceStub)

		// Flush the captured response to the real writer.
		clearFileMissing(rec)
		rec.flush(w)
		return
	}

	// No route matched — proxy to upstream.
	if h.rp == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "no matching route and no --target configured",
		})
		return
	}

	// Rewrite the request path to strip the api prefix before forwarding.
	if h.apiPrefix != "" && strings.HasPrefix(r.URL.Path, h.apiPrefix) {
		stripped := r.URL.Path[len(h.apiPrefix):]
		if stripped == "" {
			stripped = "/"
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = stripped
		if r2.URL.RawPath != "" {
			r2.URL.RawPath = stripped
		}
		r = r2
	}

	forwardRequest(w, r, h.rp, bodyBytes, h.rec)
}

// resolveCase determines which case to serve for a matched route.
// Priority order:
//  1. Conditions — first matching condition wins
//  2. Transitions — time-based sequence (if defined and no condition matched)
//  3. Fallback
func (h *Handler) resolveCase(route *config.Route, r *http.Request, bodyBytes []byte) string {
	// 1. Conditions take priority.
	for _, cond := range route.Conditions {
		if evalCondition(cond, r, bodyBytes) {
			return cond.Case
		}
	}

	// 2. Transitions — resolve by elapsed time since first request.
	if len(route.Transitions) > 0 {
		if caseName := h.transitions.resolve(route); caseName != "" {
			return caseName
		}
	}

	// 3. Fallback.
	return route.Fallback
}

// responseBuffer is a lightweight http.ResponseWriter that buffers the response
// so we can inspect it before flushing (e.g. to detect missing dynamic files).
type responseBuffer struct {
	header http.Header
	status int
	body   []byte
}

func newResponseRecorder(w http.ResponseWriter) *responseBuffer {
	return &responseBuffer{
		header: w.Header(),
		status: http.StatusOK,
	}
}

func (rb *responseBuffer) Header() http.Header  { return rb.header }
func (rb *responseBuffer) WriteHeader(code int) { rb.status = code }
func (rb *responseBuffer) Write(b []byte) (int, error) {
	rb.body = append(rb.body, b...)
	return len(b), nil
}

func (rb *responseBuffer) flush(w http.ResponseWriter) {
	w.WriteHeader(rb.status)
	_, _ = w.Write(rb.body)
}
