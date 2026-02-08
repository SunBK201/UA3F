package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sunbk201/ua3f/internal/config"
	applog "github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/server"
)

type APIServer struct {
	version        string
	cfg            *config.Config
	addr           string
	server         server.Server
	httpServer     *http.Server
	logBroadcaster *applog.Broadcaster
}

func New(addr string, version string, cfg *config.Config, srv server.Server, lb *applog.Broadcaster) *APIServer {
	return &APIServer{
		version:        version,
		cfg:            cfg,
		addr:           addr,
		server:         srv,
		logBroadcaster: lb,
	}
}

func (s *APIServer) Start() error {
	r := chi.NewRouter()

	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	if s.cfg.APIServerSecret != "" {
		r.Use(s.authMiddleware)
	}

	// api routes
	r.Get("/version", s.handleVersion)
	r.Get("/config", s.handleConfig)

	r.Get("/rules", s.handleRules)
	r.Get("/rules/header", s.handleHeaderRules)
	r.Get("/rules/body", s.handleBodyRules)
	r.Get("/rules/redirect", s.handleRedirectRules)

	r.Get("/logs", s.handleLogs)

	// pprof routes
	r.Route("/debug/pprof", func(r chi.Router) {
		r.HandleFunc("/", pprof.Index)
		r.HandleFunc("/cmdline", pprof.Cmdline)
		r.HandleFunc("/profile", pprof.Profile)
		r.HandleFunc("/symbol", pprof.Symbol)
		r.HandleFunc("/trace", pprof.Trace)
		r.Handle("/goroutine", pprof.Handler("goroutine"))
		r.Handle("/heap", pprof.Handler("heap"))
		r.Handle("/allocs", pprof.Handler("allocs"))
		r.Handle("/threadcreate", pprof.Handler("threadcreate"))
		r.Handle("/block", pprof.Handler("block"))
		r.Handle("/mutex", pprof.Handler("mutex"))
	})

	s.httpServer = &http.Server{
		Addr:              s.addr,
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("api-server listen failed: %w", err)
	}

	slog.Info("api-server started", slog.String("addr", s.addr))

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("api-server error", slog.Any("error", err))
		}
	}()

	return nil
}

func (s *APIServer) Close() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	slog.Info("api-server shutting down")
	return s.httpServer.Shutdown(ctx)
}

func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("api-server request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.String("user-agent", r.UserAgent()),
		)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
	})
}

func (s *APIServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if auth := r.Header.Get("Authorization"); auth != "" {
			if len(auth) > 7 && auth[:7] == "Bearer " {
				token = auth[7:]
			} else {
				token = auth
			}
		}
		if token == "" {
			token = r.URL.Query().Get("secret")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIServerSecret)) != 1 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
