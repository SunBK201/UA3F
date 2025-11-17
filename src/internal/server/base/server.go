package base

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	Cfg      *config.Config
	Rewriter *rewrite.Rewriter
	Cache    *expirable.LRU[string, struct{}]
}

func (s *Server) ServeConnLink(connLink *ConnLink) {
	slog.Info(fmt.Sprintf("New connection link: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)
	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol:  sniff.TCP,
		SrcAddr:   connLink.LAddr,
		DestAddr:  connLink.RAddr,
		StartTime: time.Now(),
	})
	defer statistics.RemoveConnection(connLink.LAddr, connLink.RAddr)
	defer slog.Info(fmt.Sprintf("Connection link closed: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)

	go connLink.CopyRL()

	if s.Cfg.RewriteMode == config.RewriteModeDirect || s.Cache.Contains(connLink.RAddr) {
		connLink.CopyLR()
	} else {
		_ = s.ProcessLR(connLink)
	}
}

func (s *Server) ProcessLR(c *ConnLink) (err error) {
	reader := bufio.NewReaderSize(c.LConn, 64*1024)

	defer func() {
		if err != nil {
			c.LogDebugf("ProcessLR: %s", err.Error())
		}
		if _, err = io.CopyBuffer(c.RConn, reader, one); err != nil {
			c.LogWarnf("Process io.CopyBuffer: %v", err)
		}
		_ = c.CloseLR()
	}()

	if strings.HasSuffix(c.RAddr, "443") {
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			s.Cache.Add(c.RAddr, struct{}{})
			c.LogInfo("TLS client hello detected")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.HTTPS,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
			return
		}
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTP(reader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		s.Cache.Add(c.RAddr, struct{}{})
		c.LogInfo("Sniff first request is not http, switch to direct forward")
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.TLS,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
		}
		return
	}

	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol: sniff.HTTP,
		SrcAddr:  c.LAddr,
		DestAddr: c.RAddr,
	})

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			statistics.AddConnection(
				&statistics.ConnectionRecord{
					Protocol: sniff.TCP,
					SrcAddr:  c.LAddr,
					DestAddr: c.RAddr,
				},
			)
			return
		}
		if !isHTTP {
			c.LogWarn("sniff subsequent request is not http, switch to direct forward")
			return
		}
		if req, err = http.ReadRequest(reader); err != nil {
			err = fmt.Errorf("http.ReadRequest: %w", err)
			return
		}

		decision := s.Rewriter.EvaluateRewriteDecision(req, c.LAddr, c.RAddr)

		if decision.Action == rule.ActionDrop {
			c.LogInfo("Request dropped by rule")
			continue
		}
		if decision.NeedCache {
			s.Cache.Add(c.RAddr, struct{}{})
		}

		if decision.ShouldRewrite() {
			req = s.Rewriter.Rewrite(req, c.LAddr, c.RAddr, decision)
		}

		if err = ForwardHTTP(c.RConn, req); err != nil {
			err = fmt.Errorf("ForwardHTTP: %w", err)
			return
		}
		if req.Header.Get("Upgrade") == "websocket" && req.Header.Get("Connection") == "Upgrade" {
			c.LogInfo("websocket upgrade detected, switch to direct proxy")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.WebSocket,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
			return
		}
	}
}

func ForwardHTTP(dst net.Conn, req *http.Request) error {
	if err := req.Write(dst); err != nil {
		return fmt.Errorf("req.Write: %w", err)
	}
	err := req.Body.Close()
	if err != nil {
		return fmt.Errorf("req.Body.Close: %w", err)
	}
	return nil
}
