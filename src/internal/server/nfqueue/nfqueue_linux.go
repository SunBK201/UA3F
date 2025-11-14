//go:build linux

package nfqueue

import (
	"fmt"
	"time"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/knftables"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
)

type Server struct {
	base.Server
	netfilter.Firewall
	nfqServer        *netfilter.NfqueueServer
	nftable          *knftables.Table
	SniffCtMarkLower uint32
	SniffCtMarkUpper uint32
	HTTPCtMark       uint32
	NotHTTPCtMark    uint32
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Cache:    expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		},
		SniffCtMarkLower: 10201,
		SniffCtMarkUpper: 10216,
		NotHTTPCtMark:    201,
		HTTPCtMark:       202,
		nfqServer: &netfilter.NfqueueServer{
			QueueNum: 10201,
		},
		nftable: &knftables.Table{
			Name:   "UA3F",
			Family: knftables.IPv4Family,
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
	err = s.Firewall.Setup(s.Cfg)
	if err != nil {
		logrus.Errorf("s.Firewall.Setup: %v", err)
		return err
	}
	return s.nfqServer.Start()
}

func (s *Server) Close() (err error) {
	err = s.Firewall.Cleanup()
	return err
}

// handlePacket processes a single NFQUEUE packet
func (s *Server) handlePacket(packet *netfilter.Packet) {
	if s.Cfg.RewriteMode == config.RewriteModeDirect || packet.TCP == nil {
		_ = s.nfqServer.Nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
		return
	}
	if s.Cache.Contains(packet.DstAddr) {
		s.sendVerdict(packet, &rewrite.RewriteResult{Modified: false, InCache: true})
		log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, "Destination in cache, direct forwrard")
		return
	}
	result := s.Rewriter.RewriteTCP(packet.TCP, packet.SrcAddr, packet.DstAddr)
	s.sendVerdict(packet, result)
}

func (s *Server) sendVerdict(packet *netfilter.Packet, result *rewrite.RewriteResult) {
	nf := s.nfqServer.Nf
	id := *packet.A.PacketID
	setMark, nextMark := s.getNextMark(packet, result)

	var newPacket []byte
	var err error

	if result.Modified {
		newPacket, err = packet.Serialize()
		if err != nil {
			_ = nf.SetVerdict(id, nfq.NfAccept)
			log.LogErrorWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("serializeIPPacket: %v", err))
			return
		}
	}

	log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("Sending verdict: modified=%v, setMark=%v, nextmark=%d", result.Modified, setMark, nextMark))
	if !result.Modified {
		if setMark {
			nf.SetVerdictWithOption(id, nfq.NfAccept, nfq.WithConnMark(nextMark))
		} else {
			_ = nf.SetVerdict(id, nfq.NfAccept)
		}
	} else {
		if setMark {
			if err := nf.SetVerdictWithOption(id, nfq.NfAccept, nfq.WithAlteredPacket(newPacket), nfq.WithConnMark(nextMark)); err != nil {
				_ = nf.SetVerdict(id, nfq.NfAccept)
				log.LogErrorWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("nf.SetVerdictWithOption: %v", err))
			}
		} else {
			if err := nf.SetVerdictWithOption(id, nfq.NfAccept, nfq.WithAlteredPacket(newPacket)); err != nil {
				_ = nf.SetVerdict(id, nfq.NfAccept)
				log.LogErrorWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("nf.SetVerdictWithOption: %v", err))
			}
		}
	}
}

func (s *Server) getNextMark(packet *netfilter.Packet, result *rewrite.RewriteResult) (setMark bool, mark uint32) {
	mark, found := packet.GetCtMark()
	if !found {
		return true, s.SniffCtMarkLower
	}
	log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("Current connmark: %d", mark))

	// should not happen
	if mark == s.NotHTTPCtMark {
		return false, 0
	}

	if mark == s.HTTPCtMark {
		return false, 0
	}

	if result.InCache {
		return true, s.NotHTTPCtMark
	}

	if result.Modified {
		return true, s.HTTPCtMark
	}

	if mark == 0 {
		return true, s.SniffCtMarkLower
	}

	if mark == s.SniffCtMarkUpper {
		s.Cache.Add(packet.DstAddr, struct{}{})
		return true, s.NotHTTPCtMark
	}

	if mark >= s.SniffCtMarkLower && mark < s.SniffCtMarkUpper {
		return true, mark + 1
	}

	return false, 0
}
