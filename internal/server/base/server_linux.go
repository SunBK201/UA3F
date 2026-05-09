//go:build linux

package base

import (
	"io"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
)

// TryOffload attempts to offload the given ConnLink's bidirectional
// forwarding to the eBPF sockmap fast-path.
//
// If drainReader is non-nil, any buffered data in it is flushed to RConn
// before activating the kernel-level redirect (to handle data already
// consumed during sniffing).
//
// Returns true if the connection was successfully offloaded.
// On success the caller should block on <-connLink.OffloadDone and NOT
// do any further userspace copy.
func (s *Server) TryOffload(c *common.ConnLink, drainReader io.Reader) bool {
	if s.Sockmap == nil {
		return false
	}

	// Drain any buffered bytes that the sniff phase already consumed
	// from the kernel socket buffer. sockmap cannot retroactively
	// process data that was already read into userspace.
	if drainReader != nil {
		type buffered interface {
			Buffered() int
		}
		if br, ok := drainReader.(buffered); ok && br.Buffered() > 0 {
			buf := make([]byte, br.Buffered())
			n, err := io.ReadFull(drainReader, buf)
			if n > 0 {
				if _, werr := c.RConn.Write(buf[:n]); werr != nil {
					slog.Warn("BPF offload: drain write error", "error", werr, "ConnLink", c)
					return false
				}
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				slog.Warn("BPF offload: drain read error", "error", err, "ConnLink", c)
				return false
			}
		}
	}

	lfd, err := c.LFD()
	if err != nil {
		slog.Warn("BPF offload: LFD error", "error", err, "ConnLink", c)
		return false
	}
	rfd, err := c.RFD()
	if err != nil {
		slog.Warn("BPF offload: RFD error", "error", err, "ConnLink", c)
		return false
	}
	lcookie, err := c.LSOCookie()
	if err != nil {
		slog.Warn("BPF offload: LSOCookie error", "error", err, "ConnLink", c)
		return false
	}
	rcookie, err := c.RSOCookie()
	if err != nil {
		slog.Warn("BPF offload: RSOCookie error", "error", err, "ConnLink", c)
		return false
	}

	if err := s.Sockmap.Add(lfd, rfd, lcookie, rcookie); err != nil {
		slog.Warn("BPF offload: b.Sockmap.Add error", "error", err, "ConnLink", c)
		return false
	}

	c.Offloaded = true
	slog.Info("BPF sockmap offload activated", "ConnLink", c)
	return true
}

func (s *Server) DeleteOffload(c *common.ConnLink) {
	if s.Sockmap == nil {
		return
	}

	lcookie, err := c.LSOCookie()
	if err != nil {
		slog.Warn("BPF delete offload: LSOCookie error", "error", err, "ConnLink", c)
		return
	}
	rcookie, err := c.RSOCookie()
	if err != nil {
		slog.Warn("BPF delete offload: RSOCookie error", "error", err, "ConnLink", c)
		return
	}
	s.Sockmap.Delete(lcookie, rcookie)
}
