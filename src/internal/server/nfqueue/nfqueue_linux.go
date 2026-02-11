//go:build linux

package nfqueue

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"sigs.k8s.io/knftables"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	base.Server
	netfilter.Firewall
	nfqServer        *base.NfqueueServer
	SniffCtMarkLower uint32
	SniffCtMarkUpper uint32
	HTTPCtMark       uint32
	NotHTTPCtMark    uint32
}

func New(cfg *config.Config, rw common.Rewriter, rc *statistics.Recorder) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:        cfg,
			Rewriter:   rw,
			Recorder:   rc,
			Cache:      expirable.NewLRU[string, struct{}](512, nil, 30*time.Minute),
			SkipIpChan: make(chan *net.IP, 512),
		},
		SniffCtMarkLower: 10201,
		SniffCtMarkUpper: 10216,
		NotHTTPCtMark:    201,
		HTTPCtMark:       202,
		nfqServer: &base.NfqueueServer{
			QueueNum: 10201,
		},
	}
	s.nfqServer.HandlePacket = s.handlePacket
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
	s.Recorder.Start()
	return s.nfqServer.Start()
}

func (s *Server) Close() error {
	err := s.Firewall.Cleanup()
	s.nfqServer.Close()
	return err
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

	newServer := New(cfg, newRewriter, s.Recorder)
	if err := newServer.Start(); err != nil {
		return nil, err
	}
	return newServer, nil
}

// handlePacket processes a single NFQUEUE packet
func (s *Server) handlePacket(packet *common.Packet) {
	if s.Cfg.RewriteMode == config.RewriteModeDirect || packet.TCP == nil {
		_ = s.nfqServer.Nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
		return
	}
	if s.Cache.Contains(packet.DstAddr) {
		s.sendVerdict(packet, &common.RewriteDecision{Modified: false, NeedCache: true})
		log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, "Destination in cache, direct forwrard")
		return
	}
	result := s.Rewriter.RewriteRequest(&common.Metadata{
		Packet: packet,
	})
	if result.NeedSkip {
		select {
		case s.SkipIpChan <- &packet.DstIP:
		default:
		}
	}
	s.sendVerdict(packet, result)
}

func (s *Server) sendVerdict(packet *common.Packet, result *common.RewriteDecision) {
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

func (s *Server) getNextMark(packet *common.Packet, result *common.RewriteDecision) (setMark bool, mark uint32) {
	if result.NeedSkip {
		return true, s.NotHTTPCtMark
	}

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

	if result.NeedCache {
		return true, s.NotHTTPCtMark
	}

	if result.Modified {
		return true, s.HTTPCtMark
	}

	if mark == 0 {
		return true, s.SniffCtMarkLower
	}

	if mark == s.SniffCtMarkUpper {
		slog.Debug("Connmark reached upper limit, marking as NotHTTP", slog.String("SrcAddr", packet.SrcAddr), slog.String("DstAddr", packet.DstAddr))
		s.Cache.Add(packet.DstAddr, struct{}{})
		return true, s.NotHTTPCtMark
	}

	if mark >= s.SniffCtMarkLower && mark < s.SniffCtMarkUpper {
		return true, mark + 1
	}

	return false, 0
}
