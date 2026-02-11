//go:build linux

package redirect

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/statistics"
	"sigs.k8s.io/knftables"
)

type Server struct {
	base.Server
	netfilter.Firewall
	listener net.Listener
	so_mark  int
}

func New(cfg *config.Config, rw common.Rewriter, rc *statistics.Recorder, middleMan *mitm.MiddleMan) *Server {
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
		so_mark: base.SO_MARK,
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

func (s *Server) Start() (err error) {
	err = s.Firewall.Setup(s.Cfg)
	if err != nil {
		slog.Error("s.Firewall.Setup", slog.Any("error", err))
		return err
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", s.Cfg.Port)
	if s.listener, err = net.Listen("tcp", listenAddr); err != nil {
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
				slog.Error("s.listener.Accept", slog.Any("error", err))
				continue
			}
			slog.Debug("Accept connection", slog.String("addr", client.RemoteAddr().String()))
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

func (s *Server) Restart(cfg *config.Config) (common.Server, error) {
	if err := s.Close(); err != nil {
		return nil, err
	}

	newRewriter, err := rewrite.New(cfg, s.Recorder)
	if err != nil {
		slog.Error("rewrite.New", slog.Any("error", err))
		return nil, err
	}

	newMiddleMan, err := mitm.NewMiddleMan(cfg)
	if err != nil {
		slog.Error("mitm.NewMiddleMan", slog.Any("error", err))
		return nil, err
	}

	newServer := New(cfg, newRewriter, s.Recorder, newMiddleMan)
	if err := newServer.Start(); err != nil {
		return nil, err
	}
	return newServer, nil
}

func (s *Server) HandleClient(client net.Conn) {
	addr, err := base.GetOriginalDstAddr(client)
	if err != nil {
		_ = client.Close()
		slog.Error("base.GetOriginalDstAddr", slog.Any("error", err), slog.String("client", client.RemoteAddr().String()))
		return
	}

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
