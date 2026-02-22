package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/bpf"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	base.Server
	server  *http.Server
	so_mark int
}

func New(cfg *config.Config, rw common.Rewriter, rc *statistics.Recorder, middleMan *mitm.MiddleMan, bpf *bpf.BPF) *Server {
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
			MiddleMan: middleMan,
			BPF:       bpf,
		},
		so_mark: base.SO_MARK,
	}
}

func (s *Server) Start() (err error) {
	var listener net.Listener
	listenAddr := fmt.Sprintf("%s:%d", s.Cfg.BindAddress, s.Cfg.Port)
	if listener, err = net.Listen("tcp", listenAddr); err != nil {
		return fmt.Errorf("lc.Listen: %w", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				s.handleTunneling(w, req)
			} else {
				s.handleHTTP(w, req)
			}
		}),
	}
	s.server = server

	s.Recorder.Start()
	go func() {
		if err := server.Serve(listener); err != nil {
			if err == http.ErrServerClosed {
				return
			} else {
				slog.Error("server.Serve", slog.Any("error", err))
			}
		}
	}()
	return nil
}

func (s *Server) Close() (err error) {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.BPF.Close()

	return s.server.Shutdown(ctx)
}

func (s *Server) Restart(cfg *config.Config) (common.Server, error) {
	newRewriter, err := rewrite.New(cfg, s.Recorder)
	if err != nil {
		slog.Error("rewrite.New", slog.Any("error", err))
		return nil, err
	}

	newMiddleMan, err := mitm.NewMiddleMan(cfg)
	if err != nil {
		slog.Error("mitm.NewMiddleMan", slog.Any("error", err))
		return nil, err
	}

	newServer := New(cfg, newRewriter, s.Recorder, newMiddleMan, s.BPF)

	if err := s.Close(); err != nil {
		slog.Error("old server shutdown error", slog.Any("error", err))
	}

	if err := newServer.Start(); err != nil {
		return nil, err
	}

	return newServer, nil
}

func (s *Server) handleHTTP(w http.ResponseWriter, req *http.Request) {
	metadata := &common.Metadata{}
	metadata.UpdateRequest(req)

	record := &statistics.ConnectionRecord{
		Protocol:  sniff.HTTP,
		SrcAddr:   metadata.SrcAddr(),
		DestAddr:  metadata.DestAddr(),
		StartTime: time.Now(),
	}
	s.Recorder.AddRecord(record)
	defer s.Recorder.RemoveRecord(record)

	slog.Info("HTTP proxy request", slog.String("srcAddr", metadata.SrcAddr()), slog.String("destAddr", metadata.DestAddr()))

	req, err := s.rewriteRequest(metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if req == nil {
		return // Redirected
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	metadata.UpdateResponse(resp)
	resp, err = s.rewriteResponse(metadata)
	if err != nil {
		return
	}

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) rewriteRequest(metadata *common.Metadata) (*http.Request, error) {
	decision := s.Rewriter.RewriteRequest(metadata)
	if decision.Action == action.DropRequestAction {
		log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Request dropped by rule")
		return nil, fmt.Errorf("request dropped by rule")
	}
	if decision.Redirect {
		return nil, nil
	}
	if decision.NeedCache {
		s.Cache.Add(metadata.DestAddr(), struct{}{})
	}
	s.Recorder.AddRecord(&statistics.PassThroughRecord{
		SrcAddr:  metadata.SrcAddr(),
		DestAddr: metadata.DestAddr(),
		UA:       metadata.UserAgent(),
	})
	return metadata.Request, nil
}

func (s *Server) rewriteResponse(metadata *common.Metadata) (*http.Response, error) {
	decision := s.Rewriter.RewriteResponse(metadata)
	if decision.Action == action.DropResponseAction {
		log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Response dropped by rule")
		return nil, fmt.Errorf("response dropped by rule")
	}
	return metadata.Response, nil
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
	s.ServeConnLink(&common.ConnLink{
		LConn:    src,
		RConn:    dest,
		LAddr:    req.RemoteAddr,
		RAddr:    destAddr,
		Protocol: sniff.TCP,
	})
}
