//go:build linux

package netlink

import (
	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

type Server struct {
	netfilter.Firewall
	cfg       *config.Config
	nfqServer *netfilter.NfqueueServer
	nftable   *knftables.Table
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		nfqServer: &netfilter.NfqueueServer{
			QueueNum: netfilter.HELPER_QUEUE,
		},
		nftable: &knftables.Table{
			Name:   "UA3F_HELPER",
			Family: knftables.InetFamily,
		},
	}
	s.nfqServer.HandlePacket = s.handlePacket
	s.Firewall = netfilter.Firewall{
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
	}
	return s
}

func (s *Server) Start() (err error) {
	err = s.Firewall.Setup(s.cfg)
	if err != nil {
		logrus.Errorf("s.Firewall.Setup: %v", err)
		return err
	}
	s.Firewall.DumpNFTables()
	s.Firewall.DumpIPTables()
	if s.cfg.SetTTL || s.cfg.DelTCPTimestamp || s.cfg.SetIPID {
		logrus.Info("Packet modification features enabled")
		go s.nfqServer.Start()
	}
	return nil
}

func (s *Server) Close() (err error) {
	err = s.Firewall.Cleanup()
	if err != nil {
		return err
	}
	return nil
}

// handlePacket processes a single NFQUEUE packet
func (s *Server) handlePacket(packet *netfilter.Packet) {
	nf := s.nfqServer.Nf

	modified := false
	if s.cfg.DelTCPTimestamp && packet.TCP != nil {
		modified = s.clearTCPTimestamp(packet.TCP) || modified
	}
	if s.cfg.SetIPID {
		modified = s.zeroIPID(packet) || modified
	}

	if modified {
		newPacket, err := packet.Serialize()
		if err != nil {
			logrus.Errorf("packet.Serialize: %v", err)
			_ = nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
			return
		}
		if err := nf.SetVerdictWithOption(*packet.A.PacketID, nfq.NfAccept, nfq.WithAlteredPacket(newPacket)); err != nil {
			logrus.Errorf("nf.SetVerdictWithOption: %v", err)
			_ = nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
		}
	} else {
		_ = nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
	}
}

// clearTCPTimestamp removes the TCP timestamp option from the TCP layer
// Returns true if the timestamp option was found and removed
func (s *Server) clearTCPTimestamp(tcp *layers.TCP) bool {
	if len(tcp.Options) == 0 {
		return false
	}

	modified := false
	newOptions := make([]layers.TCPOption, 0, len(tcp.Options))

	for _, opt := range tcp.Options {
		// TCP Timestamp option kind is 8
		if opt.OptionType == 8 {
			modified = true
			continue
		}
		newOptions = append(newOptions, opt)
	}
	if modified {
		tcp.Options = newOptions
	}
	return modified
}

// zeroIPID sets the IP ID field to zero for IPv4 packets
// Returns true if the packet was modified
func (s *Server) zeroIPID(packet *netfilter.Packet) bool {
	if packet.IsIPv6 {
		return false
	}
	ip4 := packet.NetworkLayer.(*layers.IPv4)
	if ip4.Id == 0 {
		return false
	}
	ip4.Id = 0
	return true
}
