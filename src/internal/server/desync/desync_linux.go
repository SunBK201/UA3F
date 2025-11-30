//go:build linux

package desync

import (
	"log/slog"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

type Server struct {
	netfilter.Firewall
	cfg       *config.Config
	nfqServer *netfilter.NfqueueServer
	CtByte    uint32
	CtPackets uint32
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		nfqServer: &netfilter.NfqueueServer{
			QueueNum: netfilter.DESYNC_QUEUE,
		},
		CtByte:    1500,
		CtPackets: 2 + 3*2,
	}
	s.nfqServer.HandlePacket = s.HandlePacket
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
	err = s.nfqServer.Start()
	if err != nil {
		return err
	}
	slog.Info("TCP Desync server started", slog.Int("ct_bytes", int(s.CtByte)), slog.Int("ct_packets", int(s.CtPackets)))
	return
}

func (s *Server) Close() error {
	err := s.Firewall.Cleanup()
	s.nfqServer.Close()
	return err
}

func (s *Server) HandlePacket(frame *netfilter.Packet) {
	fragment := s.cfg.TCPDesync.Enabled
	if frame.TCP == nil || len(frame.TCP.Payload) <= 1 || frame.TCP.FIN {
		fragment = false
	}
	s.sendVerdict(frame, fragment)
}

func (s *Server) sendVerdict(packet *netfilter.Packet, fragment bool) {
	nf := s.nfqServer.Nf
	id := *packet.A.PacketID

	if !fragment {
		_ = nf.SetVerdict(id, nfq.NfAccept)
		return
	}

	newPacket, err := packet.SerializeWithDesync()
	if err != nil {
		_ = nf.SetVerdict(id, nfq.NfAccept)
		slog.Error("packet.SerializeWithDesync", slog.Any("error", err))
		return
	}

	if err := nf.SetVerdictWithOption(id, nfq.NfAccept, nfq.WithAlteredPacket(newPacket)); err != nil {
		_ = nf.SetVerdict(id, nfq.NfAccept)
		slog.Error("nf.SetVerdictWithOption", slog.Any("error", err))
	}
}
