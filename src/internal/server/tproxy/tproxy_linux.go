//go:build linux

package tproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/utils"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	cfg      *config.Config
	rw       *rewrite.Rewriter
	listener net.Listener
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	return &Server{
		cfg: cfg,
		rw:  rw,
	}
}

func (s *Server) Start() error {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			c.Control(func(fd uintptr) {
				if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); e != nil {
					err = fmt.Errorf("unix.SetsockoptInt SO_REUSEADDR: %v", e)
					return
				}
				if e := unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_TRANSPARENT, 1); e != nil {
					err = fmt.Errorf("unix.SetsockoptInt IP_TRANSPARENT: %v", e)
					return
				}
			})
			return err
		},
	}

	var err error
	if s.listener, err = lc.Listen(context.TODO(), "tcp", s.cfg.ListenAddr); err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	go statistics.StartRecorder()

	var client net.Conn
	for {
		if client, err = s.listener.Accept(); err != nil {
			if errors.Is(err, syscall.EMFILE) {
				time.Sleep(time.Second)
			}
			logrus.Error("s.listener.Accept:", err)
			continue
		}
		logrus.Debugf("Accept connection from %s", client.RemoteAddr().String())
		go s.HandleClient(client)
	}
}

func (s *Server) HandleClient(client net.Conn) {
	addr, err := getOriginalDstAddr(client)
	if err != nil {
		_ = client.Close()
		logrus.Errorf("getOriginalDstAddr: %v", err)
		return
	}
	logrus.Debugf("Original destination address: %s", addr)

	target, err := utils.ConnectWithMark(addr, utils.SO_MARK)
	if err != nil {
		_ = client.Close()
		logrus.Warnf("utils.ConnectWithMark %s: %v", addr, err)
		return
	}

	s.ForwardTCP(client, target, addr)
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

// getOriginalDstAddr retrieves the original destination address of a TProxy connection.
func getOriginalDstAddr(conn net.Conn) (addr string, err error) {
	fd, err := utils.GetConnFD(conn)
	if err != nil {
		return "", fmt.Errorf("utils.GetConnFD: %v", err)
	}
	raw, err := unix.GetsockoptIPv6Mreq(fd, unix.SOL_IP, unix.SO_ORIGINAL_DST)
	if err != nil {
		return "", fmt.Errorf("unix.GetsockoptIPv6Mreq SO_ORIGINAL_DST: %v", err)
	}

	ip := net.IPv4(raw.Multiaddr[4], raw.Multiaddr[5], raw.Multiaddr[6], raw.Multiaddr[7])
	port := uint16(raw.Multiaddr[2])<<8 + uint16(raw.Multiaddr[3])
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}
