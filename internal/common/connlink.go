package common

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"syscall"

	"github.com/sunbk201/ua3f/internal/sniff"
)

type ConnLink struct {
	LConn    net.Conn
	RConn    net.Conn
	Metadata *Metadata

	SniffDone *sync.WaitGroup // For waiting ProcessLR First Sniff
	SniffOnce sync.Once       // Ensures SniffDone.Done() is called only once
	Protocol  sniff.Protocol

	LAddr string
	RAddr string

	lcookie uint64 // BPF cookie for L side
	rcookie uint64 // BPF cookie for R side

	Skipped   bool
	Offloaded bool // whether this ConnLink is offloaded to BPF sockmap
}

var one = make([]byte, 1)

func (c *ConnLink) DoneSniff() {
	if c.SniffDone != nil {
		c.SniffOnce.Do(func() {
			c.SniffDone.Done()
		})
	}
}

func (c *ConnLink) LIP() string {
	if tcpAddr, ok := c.LConn.RemoteAddr().(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	return ""
}

func (c *ConnLink) RIP() string {
	if tcpAddr, ok := c.RConn.RemoteAddr().(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	return ""
}

func (c *ConnLink) LPort() string {
	if tcpAddr, ok := c.LConn.RemoteAddr().(*net.TCPAddr); ok {
		return fmt.Sprintf("%d", tcpAddr.Port)
	}
	return ""
}

func (c *ConnLink) RPort() string {
	if tcpAddr, ok := c.RConn.RemoteAddr().(*net.TCPAddr); ok {
		return fmt.Sprintf("%d", tcpAddr.Port)
	}
	return ""
}

func (c *ConnLink) LFD() (int, error) {
	fd, err := sockFd(c.LConn)
	if err != nil {
		return 0, err
	}
	return int(fd), nil
}

func (c *ConnLink) RFD() (int, error) {
	fd, err := sockFd(c.RConn)
	if err != nil {
		return 0, err
	}
	return int(fd), nil
}

func (c *ConnLink) CopyLR() {
	defer func() {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.LConn.Close()
		}
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.RConn.Close()
		}
	}()
	n, _ := io.CopyBuffer(c.RConn, c.LConn, one)
	slog.Debug("CopyLR done", slog.Int64("bytes", n), slog.Any("ConnLink", c))
}

func (c *ConnLink) CopyRL() {
	defer func() {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.RConn.Close()
		}
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.LConn.Close()
		}
	}()
	n, _ := io.CopyBuffer(c.LConn, c.RConn, one)
	slog.Debug("CopyRL done", slog.Int64("bytes", n), slog.Any("ConnLink", c))
}

func (c *ConnLink) CloseLR() error {
	if c.LConn != nil {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.LConn.Close()
		}
	}
	if c.RConn != nil {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.RConn.Close()
		}
	}
	return nil
}

func (c *ConnLink) CloseRL() error {
	if c.RConn != nil {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.RConn.Close()
		}
	}
	if c.LConn != nil {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.LConn.Close()
		}
	}
	return nil
}

func (c *ConnLink) Close() error {
	if c.LConn != nil {
		_ = c.LConn.Close()
	}
	if c.RConn != nil {
		_ = c.RConn.Close()
	}
	return nil
}

func (c *ConnLink) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("LAddr", c.LAddr),
		slog.String("RAddr", c.RAddr),
	)
}

// sockFd returns the underlying OS descriptor for conn.
//   - Unix: file descriptor (fd)
//   - Windows: SOCKET handle
//
// The returned uintptr is only valid while conn is alive.
// Do NOT close it yourself.
func sockFd(conn net.Conn) (uintptr, error) {
	for i := 0; i < 8 && conn != nil; i++ {
		if sc, ok := conn.(syscall.Conn); ok {
			rc, err := sc.SyscallConn()
			if err != nil {
				return 0, err
			}

			var fd uintptr
			if err := rc.Control(func(u uintptr) {
				fd = u
			}); err != nil {
				return 0, err
			}
			return fd, nil
		}

		type netConner interface{ NetConn() net.Conn }
		if nc, ok := conn.(netConner); ok {
			next := nc.NetConn()
			if next == conn {
				break
			}
			conn = next
			continue
		}

		break
	}

	return 0, fmt.Errorf("conn type %T does not expose syscall.Conn/SyscallConn", conn)
}
