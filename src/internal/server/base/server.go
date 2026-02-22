package base

import (
	"bufio"
	"errors"
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
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	Cfg             *config.Config
	Rewriter        common.Rewriter
	Recorder        *statistics.Recorder
	Cache           *expirable.LRU[string, struct{}]
	SkipIpChan      chan *net.IP
	BufioReaderPool sync.Pool
	MiddleMan       *mitm.MiddleMan
	BPF             *bpf.BPF
}

func (s *Server) GetRewriter() common.Rewriter {
	return s.Rewriter
}

var one = make([]byte, 1)

func (s *Server) ServeConnLink(connLink *common.ConnLink) {
	slog.Info(fmt.Sprintf("New connection link: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)
	defer slog.Info(fmt.Sprintf("Connection link closed: %s <-> %s", connLink.LAddr, connLink.RAddr), "ConnLink", connLink)

	record := &statistics.ConnectionRecord{
		Protocol:  sniff.TCP,
		SrcAddr:   connLink.LAddr,
		DestAddr:  connLink.RAddr,
		StartTime: time.Now(),
	}
	s.Recorder.AddRecord(record)
	defer s.Recorder.RemoveRecord(record)

	defer func() {
		if connLink.Offloaded {
			s.BPF.DeleteOffload(connLink)
		}
	}()

	connLink.Metadata = &common.Metadata{
		ConnLink: connLink,
	}

	switch s.Cfg.RewriteMode {
	case config.RewriteModeDirect:
		s.BPF.TryOffload(connLink, nil)
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
		_ = s.ProcessLR(connLink)
	default:
		go connLink.CopyRL()
		connLink.CopyLR()
	}
}

func (s *Server) ProcessLR(c *common.ConnLink) (err error) {
	var (
		sniffReader    *bufio.Reader
		transferReader *bufio.Reader
	)

	sniffReader = s.BufioReaderPool.Get().(*bufio.Reader)
	sniffReader.Reset(c.LConn)

	defer func() {
		sniffReader.Reset(nil)
		s.BufioReaderPool.Put(sniffReader)
		if transferReader != nil && transferReader != sniffReader {
			transferReader.Reset(nil)
			s.BufioReaderPool.Put(transferReader)
		}
	}()

	defer func() {
		c.DoneSniff()

		if err != nil {
			c.LogDebugf("ProcessLR: %s", err.Error())
		}
		if c.Skipped {
			// used by reject and firewall skip
			_ = c.CloseLR()
			return
		}
		if transferReader == nil {
			transferReader = sniffReader
		}
		if _, err = io.CopyBuffer(c.RConn, transferReader, one); err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.LogDebugf("ProcessRL io.CopyBuffer: %v", err)
			} else {
				c.LogWarnf("ProcessRL io.CopyBuffer: %v", err)
			}
		}
		_ = c.CloseLR()
	}()

	if isTLS, _ := sniff.SniffTLS(sniffReader); isTLS {
		s.Cache.Add(c.RAddr, struct{}{})
		c.LogInfo("TLS client hello detected")
		c.Protocol = sniff.TLS
		s.Recorder.AddRecord(&statistics.ConnectionRecord{
			Protocol: sniff.TLS,
			SrcAddr:  c.LAddr,
			DestAddr: c.RAddr,
		})

		// If MitM is enabled, intercept the TLS connection
		if s.MiddleMan != nil {
			var tlsInfo *sniff.TLSInfo
			tlsInfo, err = sniff.SniffTLSClientHello(sniffReader)
			if err != nil {
				err = fmt.Errorf("sniff.SniffTLSClientHello: %w", err)
				return
			}
			serverName := ""
			if tlsInfo != nil && tlsInfo.ServerName != "" {
				serverName = tlsInfo.ServerName
			} else {
				return // No SNI, skip MitM
			}
			mitmDone, mitmErr := s.MiddleMan.HandleTLS(c, sniffReader, serverName)
			if mitmErr != nil {
				c.LogWarnf("MitM HandleTLS error: %v", mitmErr)
			}

			if mitmDone {
				transferReader = s.BufioReaderPool.Get().(*bufio.Reader)
				transferReader.Reset(c.LConn)
			} else {
				// MitM decided not to intercept, use the original sniffReader for transfer tls
				transferReader = sniffReader
				return
			}
		}
	}

	if transferReader == nil {
		transferReader = sniffReader // No MitM, use the sniffReader for transfer
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTPRequest(transferReader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		s.Cache.Add(c.RAddr, struct{}{})
		s.BPF.TryOffload(c, transferReader)
		c.LogInfo("Sniff first request is not http, switch to direct forward")
		return
	}

	protocol := sniff.HTTP
	if c.Protocol == sniff.TLS {
		protocol = sniff.HTTPS
	}
	c.Protocol = protocol
	s.Recorder.AddRecord(&statistics.ConnectionRecord{
		Protocol: protocol,
		SrcAddr:  c.LAddr,
		DestAddr: c.RAddr,
	})
	c.DoneSniff()

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(transferReader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			if c.Protocol == sniff.HTTPS {
				c.Protocol = sniff.TLS
			} else {
				c.Protocol = sniff.TCP
			}
			s.Recorder.AddRecord(
				&statistics.ConnectionRecord{
					Protocol: c.Protocol,
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

		if req, err = http.ReadRequest(transferReader); err != nil {
			err = fmt.Errorf("http.ReadRequest: %w", err)
			return
		}

		c.Metadata.UpdateRequest(req)

		decision := s.Rewriter.RewriteRequest(c.Metadata)
		if decision.Redirect {
			continue
		}

		switch decision.Action {
		case action.DropRequestAction:
			continue
		case action.RejectRequestAction:
			c.Skipped = true
			return
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
	defer func() {
		reader.Reset(nil)
		s.BufioReaderPool.Put(reader)
	}()

	defer func() {
		if err != nil {
			c.LogDebugf("ProcessRL: %s", err.Error())
		}
		if c.Skipped {
			// used by reject and firewall skip
			_ = c.CloseRL()
			return
		}
		if _, err = io.CopyBuffer(c.LConn, reader, one); err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.LogDebugf("ProcessRL io.CopyBuffer: %v", err)
			} else {
				c.LogWarnf("ProcessRL io.CopyBuffer: %v", err)
			}
		}
		_ = c.CloseRL()
	}()

	if c.SniffDone != nil {
		c.SniffDone.Wait()
		if c.Protocol != sniff.HTTP && c.Protocol != sniff.HTTPS {
			reader.Reset(c.RConn)
			return
		}
	}

	reader.Reset(c.RConn)

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
			c.LogWarn("sniff subsequent response is not http, switch to direct forward")
			return
		}

		if c.Protocol != sniff.HTTP && c.Protocol != sniff.HTTPS {
			return
		}

		if resp, err = http.ReadResponse(reader, nil); err != nil {
			err = fmt.Errorf("http.ReadResponse: %w", err)
			return
		}

		c.Metadata.UpdateResponse(resp)

		decision := s.Rewriter.RewriteResponse(c.Metadata)
		switch decision.Action {
		case action.DropResponseAction:
			continue
		case action.RejectResponseAction:
			c.Skipped = true
			return
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
