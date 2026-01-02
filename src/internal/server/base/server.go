package base

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	Cfg             *config.Config
	Rewriter        rewrite.Rewriter
	Recorder        *statistics.Recorder
	Cache           *expirable.LRU[string, struct{}]
	SkipIpChan      chan *net.IP
	BufioReaderPool sync.Pool
}

var one = make([]byte, 1)

func (s *Server) ServeConnLink(connLink *common.ConnLink) {
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

	connLink.Metadata = &common.Metadata{
		ConnLink: connLink,
	}

	switch s.Cfg.RewriteMode {
	case config.RewriteModeDirect:
		go connLink.CopyRL()
		connLink.CopyLR()
	case config.RewriteModeGlobal:
		go connLink.CopyRL()
		if s.Cache.Contains(connLink.RAddr) {
			connLink.CopyLR()
		} else {
			_ = s.ProcessLR(connLink)
		}
	case config.RewriteModeRule:
		if s.Rewriter.ServeResponse() {
			connLink.SniffDone = &sync.WaitGroup{}
			connLink.SniffDone.Add(1)
			go func() {
				_ = s.ProcessRL(connLink)
			}()
		} else {
			go connLink.CopyRL()
		}
		if s.Rewriter.ServeRequest() {
			_ = s.ProcessLR(connLink)
		} else {
			connLink.CopyLR()
		}
	default:
		go connLink.CopyRL()
		connLink.CopyLR()
	}
}

func (s *Server) ProcessLR(c *common.ConnLink) (err error) {
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
			c.LogWarnf("ProcessLR io.CopyBuffer: %v", err)
		}
		_ = c.CloseLR()
	}()

	if c.RPort() == "443" {
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			s.Cache.Add(c.RAddr, struct{}{})
			c.LogInfo("TLS client hello detected")
			c.Protocol = sniff.HTTPS
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
				Protocol: sniff.HTTPS,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
			return
		}
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTPRequest(reader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		s.Cache.Add(c.RAddr, struct{}{})
		c.LogInfo("Sniff first request is not http, switch to direct forward")
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			c.Protocol = sniff.TLS
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
				Protocol: sniff.TLS,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
		}
		return
	}

	if s.Cfg.RewriteMode == config.RewriteModeRule && s.Rewriter.ServeResponse() {
		c.SniffDone.Done()
	}

	c.Protocol = sniff.HTTP
	s.Recorder.AddRecord(&statistics.ConnectionRecord{
		Protocol: sniff.HTTP,
		SrcAddr:  c.LAddr,
		DestAddr: c.RAddr,
	})

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			c.Protocol = sniff.TCP
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

		c.Metadata.UpdateRequest(req)

		decision := s.Rewriter.RewriteRequest(c.Metadata)
		if decision.Action == action.DropRequestAction {
			continue
		}
		if decision.NeedCache {
			s.Cache.Add(c.RAddr, struct{}{})
		}
		if decision.NeedSkip {
			s.TrySkip(c)
		}

		s.Recorder.AddRecord(&statistics.PassThroughRecord{
			SrcAddr:  c.Metadata.SrcAddr(),
			DestAddr: c.Metadata.DestAddr(),
			UA:       c.Metadata.UserAgent(),
		})

		if err := c.Metadata.Request.Write(c.RConn); err != nil {
			return fmt.Errorf("Request.Write: %w", err)
		}

		if req.Header.Get("Upgrade") == "websocket" && req.Header.Get("Connection") == "Upgrade" {
			c.LogInfo("websocket upgrade detected, switch to direct forward")
			c.Protocol = sniff.WebSocket
			s.Recorder.AddRecord(&statistics.ConnectionRecord{
				Protocol: sniff.WebSocket,
				SrcAddr:  c.LAddr,
				DestAddr: c.RAddr,
			})
			return
		}

		if c.Skipped {
			return
		}
	}
}

func (s *Server) ProcessRL(c *common.ConnLink) (err error) {
	reader := s.BufioReaderPool.Get().(*bufio.Reader)
	reader.Reset(c.RConn)
	defer func() {
		reader.Reset(nil)
		s.BufioReaderPool.Put(reader)
	}()

	defer func() {
		if err != nil {
			c.LogDebugf("ProcessRL: %s", err.Error())
		}
		if _, err = io.CopyBuffer(c.LConn, reader, one); err != nil {
			c.LogWarnf("ProcessRL io.CopyBuffer: %v", err)
		}
		_ = c.CloseRL()
	}()

	if c.RPort() == "443" {
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			return
		}
	}

	c.SniffDone.Wait()
	if c.Protocol != sniff.HTTP {
		return
	}

	var (
		isHTTP bool
		resp   *http.Response
	)

	for {
		if isHTTP, err = sniff.SniffHTTPResponse(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPResponse: %w", err)
			return
		}
		if !isHTTP {
			c.LogWarn("sniff subsequent request is not http, switch to direct forward")
			return
		}

		if c.Protocol != sniff.HTTP {
			return
		}

		if resp, err = http.ReadResponse(reader, nil); err != nil {
			err = fmt.Errorf("http.ReadResponse: %w", err)
			return
		}

		c.Metadata.UpdateResponse(resp)

		if decision := s.Rewriter.RewriteResponse(c.Metadata); decision.Action == action.DropResponseAction {
			continue
		}

		if err := c.Metadata.Response.Write(c.LConn); err != nil {
			return fmt.Errorf("Response.Write: %w", err)
		}
	}
}

func (s *Server) TrySkip(c *common.ConnLink) {
	if c.Skipped {
		return
	}
	if s.SkipIpChan == nil {
		return
	}
	select {
	case s.SkipIpChan <- &c.RConn.RemoteAddr().(*net.TCPAddr).IP:
		c.Skipped = true
	default:
	}
}
