//go:build linux

package redirect

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/utils"
	"github.com/sunbk201/ua3f/internal/statistics"
	"golang.org/x/sys/unix"
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

func (s *Server) Start() (err error) {
	if s.listener, err = net.Listen("tcp", s.cfg.ListenAddr); err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	go statistics.StartRecorder()

	var client net.Conn
	for {
		if client, err = s.listener.Accept(); err != nil {
			logrus.Error("Accept failed: ", err)
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
		logrus.Errorf("Get original dst addr failed: %v", err)
		return
	}
	logrus.Debugf("Original destination address: %s", addr)

	target, err := utils.ConnectWithMark(addr, utils.SO_MARK)
	if err != nil {
		_ = client.Close()
		logrus.Errorf("Dial target %s failed: %v", addr, err)
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

	// Client -> Server (rewriter)
	go utils.ProxyHalf(target, client, s.rw, destAddr)
}

// getOriginalDstAddr retrieves the original destination address of the redirected connection.
func getOriginalDstAddr(conn net.Conn) (addr string, err error) {
	fd, err := utils.GetConnFD(conn)
	if err != nil {
		return "", fmt.Errorf("failed to get file descriptor: %v", err)
	}
	raw, err := unix.GetsockoptIPv6Mreq(fd, unix.SOL_IP, unix.SO_ORIGINAL_DST)
	if err != nil {
		return "", fmt.Errorf("getsockopt SO_ORIGINAL_DST failed: %v", err)
	}

	ip := net.IPv4(raw.Multiaddr[4], raw.Multiaddr[5], raw.Multiaddr[6], raw.Multiaddr[7])
	port := uint16(raw.Multiaddr[2])<<8 + uint16(raw.Multiaddr[3])
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}
