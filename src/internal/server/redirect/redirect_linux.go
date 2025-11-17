//go:build linux

package redirect

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
	"sigs.k8s.io/knftables"
)

type Server struct {
	base.Server
	netfilter.Firewall
	listener net.Listener
	so_mark  int
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Cache:    expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		},
		so_mark: netfilter.SO_MARK,
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

func (s *Server) Start() (err error) {
	err = s.Firewall.Setup(s.Cfg)
	if err != nil {
		slog.Error("s.Firewall.Setup", slog.Any("error", err))
		return err
	}
	if s.listener, err = net.Listen("tcp", s.Cfg.ListenAddr); err != nil {
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
			slog.Error("s.listener.Accept", slog.Any("error", err))
			continue
		}
		slog.Debug("Accept connection", slog.String("addr", client.RemoteAddr().String()))
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
		slog.Error("base.GetOriginalDstAddr", slog.Any("error", err))
		return
	}

	target, err := base.ConnectWithMark(addr, s.so_mark)
	if err != nil {
		_ = client.Close()
		slog.Warn("base.ConnectWithMark", slog.String("addr", addr), slog.Any("error", err))
		return
	}

	s.ServeConnLink(&base.ConnLink{
		LConn: client,
		RConn: target,
		LAddr: client.RemoteAddr().String(),
		RAddr: addr,
	})
}
