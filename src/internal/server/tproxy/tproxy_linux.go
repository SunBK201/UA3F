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

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
)

type Server struct {
	netfilter.Firewall
	base.Server
	listener         net.Listener
	so_mark          int
	tproxyFwMark     string
	tproxyRouteTable string
	ignoreMark       []string
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Cache:    expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		},
		so_mark:          netfilter.SO_MARK,
		tproxyFwMark:     "0x1c9",
		tproxyRouteTable: "0x1c9",
		ignoreMark: []string{
			"0x162",
			"0x1ed4", // sc tproxy mark 7892
		},
	}
	s.Firewall = netfilter.Firewall{
		Nftable: &knftables.Table{
			Name:   "UA3F",
			Family: knftables.IPv4Family,
		},
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
	}
	return s
}

func (s *Server) Start() error {
	var err error

	err = s.Firewall.Setup(s.Cfg)
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

	if s.listener, err = lc.Listen(context.TODO(), "tcp", s.Cfg.ListenAddr); err != nil {
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
	addr, err := base.GetOriginalDstAddr(client)
	if err != nil {
		_ = client.Close()
		logrus.Errorf("base.GetOriginalDstAddr: %v", err)
		return
	}

	target, err := base.ConnectWithMark(addr, s.so_mark)
	if err != nil {
		_ = client.Close()
		logrus.Warnf("base.ConnectWithMark %s: %v", addr, err)
		return
	}

	s.ServeConnLink(&base.ConnLink{
		LConn: client,
		RConn: target,
		LAddr: client.RemoteAddr().String(),
		RAddr: addr,
	})
}
