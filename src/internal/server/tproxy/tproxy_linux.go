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
	"sigs.k8s.io/knftables"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/utils"
)

type Server struct {
	netfilter.Firewall
	cfg              *config.Config
	rw               *rewrite.Rewriter
	listener         net.Listener
	so_mark          int
	tproxyFwMark     string
	tproxyRouteTable string
	nftable          *knftables.Table
	ignoreMark       []string
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		cfg:              cfg,
		rw:               rw,
		so_mark:          netfilter.SO_MARK,
		tproxyFwMark:     "0x1c9",
		tproxyRouteTable: "0x1c9",
		nftable: &knftables.Table{
			Name:   "UA3F",
			Family: knftables.IPv4Family,
		},
		ignoreMark: []string{
			"0x162",
			"0x1ed4", // sc tproxy mark 7892
		},
	}
	s.Firewall = netfilter.Firewall{
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
	}
	return s
}

func (s *Server) Start() error {
	var err error

	err = s.Firewall.Setup(s.cfg)
	if err != nil {
		logrus.Errorf("s.Firewall.Setup: %v", err)
		return err
	}
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

	if s.listener, err = lc.Listen(context.TODO(), "tcp", s.cfg.ListenAddr); err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}
	var client net.Conn
	for {
		if client, err = s.listener.Accept(); err != nil {
			if errors.Is(err, syscall.EMFILE) {
				time.Sleep(time.Second)
			} else if errors.Is(err, net.ErrClosed) {
				return nil
			}
			logrus.Error("s.listener.Accept:", err)
			continue
		}
		logrus.Debugf("Accept connection from %s", client.RemoteAddr().String())
		go s.HandleClient(client)
	}
}

func (s *Server) Close() error {
	_ = s.Firewall.Cleanup()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) HandleClient(client net.Conn) {
	addr, err := getOriginalDstAddr(client)
	if err != nil {
		_ = client.Close()
		logrus.Errorf("getOriginalDstAddr: %v", err)
		return
	}
	logrus.Debugf("Original destination address: %s", addr)

	target, err := utils.ConnectWithMark(addr, s.so_mark)
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
