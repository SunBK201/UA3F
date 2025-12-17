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
	CtByte           uint32
	CtPackets        uint32
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		ReorderNfqServer: &base.NfqueueServer{
			QueueNum: netfilter.DESYNC_QUEUE,
		},
		CtByte:    1500,
		CtPackets: 2 + 3*2,
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
	if s.cfg.TCPDesync.Bytes > 0 {
		s.CtByte = s.cfg.TCPDesync.Bytes
	}
	if s.cfg.TCPDesync.Packets > 0 {
		s.CtPackets = s.cfg.TCPDesync.Packets
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
	slog.Info("TCP Desync server started", slog.Int("ct_bytes", int(s.CtByte)), slog.Int("ct_packets", int(s.CtPackets)))
	return
}

func (s *Server) Close() error {
	err := s.Firewall.Cleanup()
	s.ReorderNfqServer.Close()
	return err
}
