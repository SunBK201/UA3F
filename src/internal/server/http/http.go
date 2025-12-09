package http

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	base.Server
	so_mark int
}

func New(cfg *config.Config, rw *rewrite.Rewriter, rc *statistics.Recorder) *Server {
	return &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Recorder: rc,
			Cache:    expirable.NewLRU[string, struct{}](512, nil, 30*time.Minute),
			BufioReaderPool: sync.Pool{
				New: func() interface{} {
					return bufio.NewReaderSize(nil, 16*1024)
				},
			},
		},
		so_mark: base.SO_MARK,
	}
}

func (s *Server) Start() (err error) {
	s.Recorder.Start()
	server := &http.Server{
		Addr: s.Cfg.ListenAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				s.handleTunneling(w, req)
			} else {
				s.handleHTTP(w, req)
			}
		}),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server.ListenAndServe", slog.Any("error", err))
		}
	}()
	return nil
}

func (s *Server) Close() (err error) {
	return nil
}

func (s *Server) handleHTTP(w http.ResponseWriter, req *http.Request) {
	destPort := req.URL.Port()
	if destPort == "" {
		destPort = "80"
	}
	destAddr := fmt.Sprintf("%s:%s", req.URL.Hostname(), destPort)

	record := &statistics.ConnectionRecord{
		Protocol:  sniff.HTTP,
		SrcAddr:   req.RemoteAddr,
		DestAddr:  destAddr,
		StartTime: time.Now(),
	}
	s.Recorder.AddRecord(record)
	defer s.Recorder.RemoveRecord(record)

	slog.Info("HTTP proxy request", slog.String("srcAddr", req.RemoteAddr), slog.String("destAddr", destAddr))

	req, err := s.rewrite(req, req.RemoteAddr, destAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) rewrite(req *http.Request, srcAddr, dstAddr string) (*http.Request, error) {
	decision := s.Rewriter.EvaluateRewriteDecision(req, srcAddr, dstAddr)
	if decision.Action == rule.ActionDrop {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Request dropped by rule")
		return nil, fmt.Errorf("request dropped by rule")
	}
	if decision.NeedCache {
		s.Cache.Add(dstAddr, struct{}{})
	}
	if decision.ShouldRewrite() {
		req = s.Rewriter.Rewrite(req, srcAddr, dstAddr, decision)
	}
	return req, nil
}

func (s *Server) handleTunneling(w http.ResponseWriter, req *http.Request) {
	slog.Info("HTTP CONNECT request", slog.String("host", req.Host))
	destAddr := req.Host
	dest, err := base.Connect(destAddr, s.so_mark)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	src, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if _, err := src.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		slog.Warn("failed to write CONNECT response to client", slog.String("client", req.RemoteAddr), slog.Any("error", err))
		_ = src.Close()
		_ = dest.Close()
		return
	}
	s.ServeConnLink(&base.ConnLink{
		LConn: src,
		RConn: dest,
		LAddr: req.RemoteAddr,
		RAddr: destAddr,
	})
}
