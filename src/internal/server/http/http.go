package http

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/server/utils"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	cfg *config.Config
	rw  *rewrite.Rewriter
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	return &Server{
		cfg: cfg,
		rw:  rw,
	}
}

func (s *Server) Start() (err error) {
	server := &http.Server{
		Addr: s.cfg.ListenAddr,
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

	logrus.Infof("HTTP request for %s", destAddr)

	target, err := utils.Connect(destAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if err := target.Close(); err != nil {
			logrus.Warnf("target.Close %s: %v", destAddr, err)
		}
	}()

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
			logrus.Warnf("resp.Body.Close %s: %v", destAddr, cerr)
		}
	}()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	statistics.RemoveConnection(req.RemoteAddr, destAddr)
}

func (s *Server) handleTunneling(w http.ResponseWriter, req *http.Request) {
	logrus.Infof("HTTP CONNECT request for %s", req.Host)
	destAddr := req.Host
	dest, err := utils.Connect(destAddr)
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
		logrus.Warnf("failed to write CONNECT response to client %s: %v", req.RemoteAddr, err)
		_ = client.Close()
		_ = dest.Close()
		return
	}
	s.ForwardTCP(client, dest, destAddr)
}

func (s *Server) rewriteAndForward(target net.Conn, req *http.Request, dstAddr, srcAddr string) (err error) {
	decision := s.rw.EvaluateRewriteDecision(req, srcAddr, dstAddr)
	if decision.Action == rule.ActionDrop {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Request dropped by rule")
		return fmt.Errorf("request dropped by rule")
	}
	if decision.ShouldRewrite() {
		req = s.rw.Rewrite(req, srcAddr, dstAddr, decision)
	}
	if err = s.rw.Forward(target, req); err != nil {
		err = fmt.Errorf("s.rw.Forward: %w", err)
		return
	}
	return nil
}

func (s *Server) HandleClient(client net.Conn) {
}

// ForwardTCP proxies traffic in both directions.
// target->client uses raw copy.
// client->target is processed by the rewriter (or raw if cached).
func (s *Server) ForwardTCP(client, target net.Conn, destAddr string) {
	// Server -> Client (raw)
	go utils.CopyHalf(client, target)

	if s.cfg.RewriteMode == config.RewriteModeDirect {
		// Client -> Server (raw)
		go utils.CopyHalf(target, client)
		return
	}
	// Client -> Server (rewriter)
	go utils.ProxyHalf(target, client, s.rw, destAddr)
}
