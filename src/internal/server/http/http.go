package http

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
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
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	return &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Cache:    expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		},
	}
}

func (s *Server) Start() (err error) {
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
	return server.ListenAndServe()
}

func (s *Server) Close() (err error) {
	return nil
}

func (s *Server) handleHTTP(w http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host
	destPort := req.URL.Port()
	if destPort == "" {
		destPort = "80"
	}
	destAddr := fmt.Sprintf("%s:%s", req.URL.Hostname(), destPort)
	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol:  sniff.HTTP,
		SrcAddr:   req.RemoteAddr,
		DestAddr:  destAddr,
		StartTime: time.Now(),
	})
	defer statistics.RemoveConnection(req.RemoteAddr, destAddr)

	target, err := base.Connect(destAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if err := target.Close(); err != nil {
			slog.Warn("target.Close", slog.String("destAddr", destAddr), slog.Any("error", err))
		}
	}()

	slog.Info("New HTTP proxy request", slog.String("srcAddr", req.RemoteAddr), slog.String("destAddr", destAddr))

	err = s.rewriteAndForward(target, req, req.Host, req.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	resp, err := http.ReadResponse(bufio.NewReader(target), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("resp.Body.Close", slog.String("destAddr", destAddr), slog.Any("error", cerr))
		}
	}()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) rewriteAndForward(target net.Conn, req *http.Request, dstAddr, srcAddr string) (err error) {
	decision := s.Rewriter.EvaluateRewriteDecision(req, srcAddr, dstAddr)
	if decision.Action == rule.ActionDrop {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Request dropped by rule")
		return fmt.Errorf("request dropped by rule")
	}
	if decision.NeedCache {
		s.Cache.Add(dstAddr, struct{}{})
	}
	if decision.ShouldRewrite() {
		req = s.Rewriter.Rewrite(req, srcAddr, dstAddr, decision)
	}
	if err = base.ForwardHTTP(target, req); err != nil {
		err = fmt.Errorf("base.ForwardHTTP: %w", err)
		return
	}
	return nil
}

func (s *Server) handleTunneling(w http.ResponseWriter, req *http.Request) {
	slog.Info("HTTP CONNECT request", slog.String("host", req.Host))
	destAddr := req.Host
	dest, err := base.Connect(destAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		slog.Warn("failed to write CONNECT response to client", slog.String("client", req.RemoteAddr), slog.Any("error", err))
		_ = client.Close()
		_ = dest.Close()
		return
	}
	s.ServeConnLink(&base.ConnLink{
		LConn: client,
		RConn: dest,
		LAddr: client.RemoteAddr().String(),
		RAddr: destAddr,
	})
}

func (s *Server) HandleClient(client net.Conn) {
}
