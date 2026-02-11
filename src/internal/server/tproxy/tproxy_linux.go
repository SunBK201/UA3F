//go:build linux

package tproxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"sigs.k8s.io/knftables"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/statistics"
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

func New(cfg *config.Config, rw rewrite.Rewriter, rc *statistics.Recorder, middleMan *mitm.MiddleMan) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:        cfg,
			Rewriter:   rw,
			Recorder:   rc,
			Cache:      expirable.NewLRU[string, struct{}](512, nil, 30*time.Minute),
			SkipIpChan: make(chan *net.IP, 512),
			BufioReaderPool: sync.Pool{
				New: func() interface{} {
					return bufio.NewReaderSize(nil, 16*1024)
				},
			},
			MiddleMan: middleMan,
		},
		so_mark:          base.SO_MARK,
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
			Family: knftables.InetFamily,
		},
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		NftWatch:   s.nftWatch,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
		IptWatch:   s.iptWatch,
	}
	return s
}

func (s *Server) Start() error {
	var err error

	err = s.Firewall.Setup(s.Cfg)
	if err != nil {
		slog.Error("s.Firewall.Setup", slog.Any("error", err))
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

	listenAddr := fmt.Sprintf("0.0.0.0:%d", s.Cfg.Port)
	if s.listener, err = lc.Listen(context.TODO(), "tcp", listenAddr); err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	s.Recorder.Start()

	go func() {
		var client net.Conn
		for {
			if client, err = s.listener.Accept(); err != nil {
				if errors.Is(err, syscall.EMFILE) {
					time.Sleep(time.Second)
				} else if errors.Is(err, net.ErrClosed) {
					return
				}
				slog.Error(fmt.Sprintf("s.listener.Accept: %v", err))
				continue
			}
			slog.Debug(fmt.Sprintf("Accept connection from %s", client.RemoteAddr().String()))
			go s.HandleClient(client)
		}
	}()
	return nil
}

func (s *Server) Close() error {
	_ = s.Firewall.Cleanup()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) HandleClient(client net.Conn) {
	addr := client.LocalAddr().String()

	target, err := base.Connect(addr, s.so_mark)
	if err != nil {
		_ = client.Close()
		slog.Warn("base.Connect", slog.String("addr", addr), slog.Any("error", err))
		return
	}

	s.ServeConnLink(&common.ConnLink{
		LConn: client,
		RConn: target,
		LAddr: client.RemoteAddr().String(),
		RAddr: addr,
	})
}
