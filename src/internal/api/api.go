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
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	applog "github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/server/desync"
	"github.com/sunbk201/ua3f/internal/server/netlink"
)

type APIServer struct {
	version        string
	cfg            *config.Config
	addr           string
	httpServer     *http.Server
	logBroadcaster *applog.Broadcaster

	Server common.Server
	Helper *netlink.Server
	Desync *desync.Server
}

func New(version string, cfg *config.Config, lb *applog.Broadcaster) *APIServer {
	return &APIServer{
		version:        version,
		cfg:            cfg,
		addr:           cfg.APIServer,
		logBroadcaster: lb,
	}
}

func (s *APIServer) Start() error {
	if s.cfg.APIServer == "" {
		return nil
	}

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

	r.Get("/restart", s.handleRestart)

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

func (s *APIServer) RestartSystem() error {
	newCfg, err := config.ReloadFromFile()
	if err != nil {
		return err
	}
	slog.Info("config reloaded successfully")

	if s.Server != nil {
		if newServer, err := s.Server.Restart(newCfg); err != nil {
			return err
		} else {
			s.Server = newServer
		}
	}
	if s.Desync != nil {
		if newDesync, err := s.Desync.Restart(newCfg); err != nil {
			return err
		} else {
			s.Desync = newDesync
		}
	}
	if s.Helper != nil {
		if newHelper, err := s.Helper.Restart(newCfg); err != nil {
			return err
		} else {
			s.Helper = newHelper
		}
	}
	slog.Info("ua3f restarted successfully")
	return nil
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
