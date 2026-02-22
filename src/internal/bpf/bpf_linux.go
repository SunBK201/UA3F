//go:build linux

package bpf

import (
	"io"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
)

type BPF struct {
	Sockmap *Sockmap
}

func NewBPF(cfg *config.Config) (*BPF, error) {
	if !cfg.BPFOffload {
		return nil, nil
	}

	sockmap, err := NewSockmap()
	if err != nil {
		return nil, err
	}

	return &BPF{
		Sockmap: sockmap,
	}, nil
}

func (b *BPF) Start() {
	if b == nil {
		return
	}
}

func (b *BPF) Close() {
	if b == nil {
		return
	}
	b.Sockmap.Close()
}

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
func (b *BPF) TryOffload(c *common.ConnLink, drainReader io.Reader) bool {
	if b == nil {
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
					c.LogWarnf("BPF offload: drain write error: %v", werr)
					return false
				}
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				c.LogWarnf("BPF offload: drain read error: %v", err)
				return false
			}
		}
	}

	lfd, err := c.LFD()
	if err != nil {
		c.LogWarnf("BPF offload: LFD: %v", err)
		return false
	}
	rfd, err := c.RFD()
	if err != nil {
		c.LogWarnf("BPF offload: RFD: %v", err)
		return false
	}
	lcookie, err := c.LSOCookie()
	if err != nil {
		c.LogWarnf("BPF offload: LSOCookie: %v", err)
		return false
	}
	rcookie, err := c.RSOCookie()
	if err != nil {
		c.LogWarnf("BPF offload: RSOCookie: %v", err)
		return false
	}

	if err := b.Sockmap.Add(lfd, rfd, lcookie, rcookie); err != nil {
		c.LogWarnf("BPF offload: b.Sockmap.Add: %v", err)
		return false
	}

	c.Offloaded = true
	c.LogInfo("BPF sockmap offload activated")
	return true
}

func (b *BPF) DeleteOffload(c *common.ConnLink) {
	if b == nil {
		return
	}
	lcookie, err := c.LSOCookie()
	if err != nil {
		c.LogWarnf("BPF delete offload: LSOCookie: %v", err)
		return
	}
	rcookie, err := c.RSOCookie()
	if err != nil {
		c.LogWarnf("BPF delete offload: RSOCookie: %v", err)
		return
	}
	b.Sockmap.Delete(lcookie, rcookie)
}
