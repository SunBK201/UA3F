//go:build linux

package desync

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/server/base"
	"sigs.k8s.io/knftables"
)

type Server struct {
	netfilter.Firewall
	cfg              *config.Config
	ReorderNfqServer *base.NfqueueServer
	ReorderByte      uint32
	ReorderPackets   uint32
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		ReorderNfqServer: &base.NfqueueServer{
			QueueNum: netfilter.DESYNC_QUEUE,
		},
		ReorderByte:    1500,
		ReorderPackets: 2 + 3*2,
	}
	s.ReorderNfqServer.HandlePacket = s.ReorderPacket
	s.Firewall = netfilter.Firewall{
		Nftable: &knftables.Table{
			Name:   "UA3F_DESYNC",
			Family: knftables.InetFamily,
		},
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
	}
	if s.cfg.TCPDesync.ReorderBytes > 0 {
		s.ReorderByte = s.cfg.TCPDesync.ReorderBytes
	}
	if s.cfg.TCPDesync.ReorderPackets > 0 {
		s.ReorderPackets = s.cfg.TCPDesync.ReorderPackets
	}
	return s
}

func (s *Server) Start() (err error) {
	err = s.Firewall.Setup(s.cfg)
	if err != nil {
		slog.Error("s.Firewall.Setup", slog.Any("error", err))
		return err
	}
	err = s.ReorderNfqServer.Start()
	if err != nil {
		return err
	}
	slog.Info("TCP Desync server started", slog.Int("reorder_bytes", int(s.ReorderByte)), slog.Int("reorder_packets", int(s.ReorderPackets)))
	return
}

func (s *Server) Close() error {
	err := s.Firewall.Cleanup()
	s.ReorderNfqServer.Close()
	return err
}
