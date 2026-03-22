package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
)

// ServerOptions holds all runtime configuration for the proxy server.
type ServerOptions struct {
	Target     string
	Port       int
	ConfigPath string
	ApiPrefix  string // stripped from request path before route matching and upstream forwarding
	RecordMode bool
}

// Server wraps the HTTP server and the config loader.
type Server struct {
	opts   ServerOptions
	loader *config.Loader
	srv    *http.Server
}

// NewServer initialises the proxy server.
func NewServer(opts ServerOptions) (*Server, error) {
	var rp *httputil.ReverseProxy
	if opts.Target != "" {
		var rpErr error
		rp, rpErr = newReverseProxy(opts.Target)
		if rpErr != nil {
			return nil, fmt.Errorf("creating reverse proxy: %w", rpErr)
		}
	}

	// transitions is created here so we can reference it in the onChange
	// callback before the handler is fully constructed.
	ts := newTransitionState()

	loader, err := config.NewLoader(opts.ConfigPath, func(cfg *config.Config) {
		logger.Info("config reloaded", "routes", len(cfg.Routes))
		ts.Reset()
	})
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	handler := NewHandlerWithTransitions(loader, rp, opts.RecordMode, opts.ApiPrefix, ts)

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	chain := corsMiddleware(logger.Middleware(mux))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      chain,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{opts: opts, loader: loader, srv: srv}, nil
}

// Loader returns the config loader, allowing other servers (e.g. gRPC) to
// share the same loaded config and config directory.
func (s *Server) Loader() *config.Loader {
	return s.loader
}

// Start listens and serves. It blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		prefixMsg := s.opts.ApiPrefix
		if prefixMsg == "" {
			prefixMsg = "(none)"
		}
		logger.Info("mockr running",
			"port", s.opts.Port,
			"config", s.opts.ConfigPath,
			"routes", len(s.loader.Get().Routes),
			"target", s.opts.Target,
			"prefix", prefixMsg,
			"record", s.opts.RecordMode,
		)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutCtx)
		s.loader.Close()
		return nil
	case err := <-errCh:
		s.loader.Close()
		return err
	}
}

// corsMiddleware injects CORS headers on every response and handles preflight.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
