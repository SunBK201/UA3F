//go:build linux

package desync

import (
	"log/slog"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/sunbk201/ua3f/internal/server/base"
)

func (s *Server) ReorderPacket(frame *base.Packet) {
	nf := s.ReorderNfqServer.Nf
	id := *frame.A.PacketID

	if frame.TCP == nil || len(frame.TCP.Payload) <= 1 || frame.TCP.FIN {
		_ = nf.SetVerdict(id, nfq.NfAccept)
		return
	}

	newPacket, err := frame.SerializeWithDesync()
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
