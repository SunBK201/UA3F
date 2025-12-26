package base

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	Cfg             *config.Config
	Rewriter        *rewrite.Rewriter
	Recorder        *statistics.Recorder
	Cache           *expirable.LRU[string, struct{}]
	SkipIpChan      chan *net.IP
	BufioReaderPool sync.Pool
}

func (s *Server) ServeConnLink(connLink *ConnLink) {
	slog.Info(fmt.Sprintf("New connection link: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)
	record := &statistics.ConnectionRecord{
		Protocol:  sniff.TCP,
		SrcAddr:   connLink.LAddr,
		DestAddr:  connLink.RAddr,
		StartTime: time.Now(),
	}
	s.Recorder.AddRecord(record)
	defer s.Recorder.RemoveRecord(record)
	defer slog.Info(fmt.Sprintf("Connection link closed: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)

	go connLink.CopyRL()

	if s.Cfg.RewriteMode == config.RewriteModeDirect || s.Cache.Contains(connLink.RAddr) {
		connLink.CopyLR()
	} else {
		_ = s.ProcessLR(connLink)
	}
}

func (s *Server) ProcessLR(c *ConnLink) (err error) {
	reader := s.BufioReaderPool.Get().(*bufio.Reader)
	reader.Reset(c.LConn)
	defer func() {
		reader.Reset(nil)
		s.BufioReaderPool.Put(reader)
	}()

	defer func() {
		if err != nil {
			c.LogDebugf("ProcessLR: %s", err.Error())
		}
		if c.Skipped {
			_ = c.CloseLR()
			return
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
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
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
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
				Protocol: sniff.TLS,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
		}
		return
	}

	s.Recorder.AddRecord(&statistics.ConnectionRecord{
		Protocol: sniff.HTTP,
		SrcAddr:  c.LAddr,
		DestAddr: c.RAddr,
	})

	var req *http.Request
	var destAddr string

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			s.Recorder.AddRecord(
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

		destAddr = req.Host
		if len(destAddr) == 0 {
			destAddr = c.RAddr
		}
		if strings.IndexByte(destAddr, ':') == -1 {
			destAddr = net.JoinHostPort(destAddr, c.RPort())
		}

		decision := s.Rewriter.EvaluateRewriteDecision(req, c.LAddr, destAddr)

		if decision.Action == action.DropAction {
			c.LogInfo("Request dropped by rule")
			continue
		}
		if decision.NeedCache {
			s.Cache.Add(c.RAddr, struct{}{})
		}
		if !c.Skipped && decision.NeedSkip && s.SkipIpChan != nil {
			select {
			case s.SkipIpChan <- &c.RConn.RemoteAddr().(*net.TCPAddr).IP:
				c.Skipped = true
			default:
			}
		}

		if decision.ShouldRewrite() {
			req = s.Rewriter.Rewrite(req, c.LAddr, destAddr, decision)
		}

		if err := req.Write(c.RConn); err != nil {
			return fmt.Errorf("req.Write: %w", err)
		}

		if req.Header.Get("Upgrade") == "websocket" && req.Header.Get("Connection") == "Upgrade" {
			c.LogInfo("websocket upgrade detected, switch to direct forward")
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
				Protocol: sniff.WebSocket,
				SrcAddr:  c.LAddr,
				DestAddr: destAddr,
			})
			return
		}

		if c.Skipped {
			return
		}
	}
}
