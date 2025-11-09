package http

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/server/utils"
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

func (s *Server) handleHTTP(w http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = req.Host
	destPort := req.URL.Port()
	if destPort == "" {
		destPort = "80"
	}
	destAddr := fmt.Sprintf("%s:%s", req.URL.Hostname(), destPort)

	logrus.Infof("HTTP request for %s", destAddr)

	target, err := utils.Connect(destAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer target.Close()

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
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	s.ForwardTCP(client, dest, destAddr)
}

func (s *Server) rewriteAndForward(target net.Conn, req *http.Request, dstAddr, srcAddr string) (err error) {
	rw := s.rw

	// 获取重写决策（只匹配一次规则）
	decision := rw.EvaluateRewriteDecision(req, srcAddr, dstAddr)

	// Handle DROP action
	if decision.Action == rule.ActionDrop {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Request dropped by rule")
		return fmt.Errorf("request dropped by rule")
	}

	// 如果需要重写，执行重写操作
	if decision.ShouldRewrite() {
		req = rw.Rewrite(req, srcAddr, dstAddr, decision)
	}

	if err = rw.Forward(target, req); err != nil {
		err = fmt.Errorf("r.forward: %w", err)
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

	if s.cfg.RewriteMode == "direct" {
		// Client -> Server (raw)
		go utils.CopyHalf(target, client)
		return
	}
	// Client -> Server (rewriter)
	go utils.ProxyHalf(target, client, s.rw, destAddr)
}
