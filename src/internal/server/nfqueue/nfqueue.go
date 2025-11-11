package nfqueue

import (
	"fmt"

	nfq "github.com/florianl/go-nfqueue/v2"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/rewrite"
)

type Server struct {
	cfg                 *config.Config
	rw                  *rewrite.Rewriter
	nfqServer           *netfilter.NfqueueServer
	SniffMarkRangeLower uint32
	SniffMarkRangeUpper uint32
	HTTPMark            uint32
	NotHTTPMark         uint32
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		cfg:                 cfg,
		rw:                  rw,
		SniffMarkRangeLower: 10201,
		SniffMarkRangeUpper: 10216,
		NotHTTPMark:         201,
		HTTPMark:            202,
		nfqServer: &netfilter.NfqueueServer{
			QueueNum: 10201,
		},
	}
	s.nfqServer.HandlePacket = s.handlePacket
	return s
}

func (s *Server) Start() (err error) {
	return s.nfqServer.Start()
}

func (s *Server) Close() (err error) {
	// err = s.nfqServer.Nf.Close()
	return nil
}

// handlePacket processes a single NFQUEUE packet
func (s *Server) handlePacket(packet *netfilter.Packet) {
	if s.cfg.RewriteMode == config.RewriteModeDirect || packet.TCP == nil {
		_ = s.nfqServer.Nf.SetVerdict(*packet.A.PacketID, nfq.NfAccept)
		return
	}
	if s.rw.Cache.Contains(packet.DstAddr) {
		s.sendVerdict(packet, &rewrite.RewriteResult{Modified: false, InCache: true})
		log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, "Destination in cache, skipping User-Agent rewrite")
		return
	}
	result := s.rw.RewriteTCP(packet.TCP, packet.SrcAddr, packet.DstAddr)
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

	log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("Sending verdict: Modified=%v, SetMark=%v, NextMark=%d", result.Modified, setMark, nextMark))
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
	mark, found := packet.GetConnMark()
	if !found {
		return true, s.SniffMarkRangeLower
	}
	log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, fmt.Sprintf("Current connmark: %d", mark))

	// should not happen
	if mark == s.NotHTTPMark {
		return false, 0
	}

	if mark == s.HTTPMark {
		return false, 0
	}

	if result.InCache {
		return true, s.NotHTTPMark
	}

	if result.InWhitelist {
		return true, s.NotHTTPMark
	}

	if result.Modified {
		return true, s.HTTPMark
	}

	if mark == 0 {
		return true, s.SniffMarkRangeLower
	}

	if mark == s.SniffMarkRangeUpper {
		s.rw.Cache.Add(packet.DstAddr, struct{}{})
		return true, s.NotHTTPMark
	}

	if mark >= s.SniffMarkRangeLower && mark < s.SniffMarkRangeUpper {
		return true, mark + 1
	}

	return false, 0
}
