//go:build linux

package desync

import (
	"fmt"
	"log/slog"
	"syscall"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sunbk201/ua3f/internal/server/base"
)

func (s *Server) InjectPacket(p *base.Packet) {
	defer func() {
		_ = s.InjectNfqServer.Nf.SetVerdict(*p.A.PacketID, nfq.NfAccept)
	}()

	if !s.checkTTL(p) {
		slog.Debug("Packet TTL too high, skipping injection", slog.String("src", p.SrcAddr), slog.String("dst", p.DstAddr))
		return
	}

	newTCP := &layers.TCP{
		SrcPort: p.TCP.DstPort, // Swap ports
		DstPort: p.TCP.SrcPort,
		Seq:     p.TCP.Ack,            // Our SEQ = their ACK
		Ack:     p.TCP.Seq + 1,        // ACK their SYN
		ACK:     true,                 // ACK flag
		PSH:     true,                 // PSH flag for data
		Window:  65535,                // Window size
		Options: []layers.TCPOption{}, // No options
	}

	// Build network layer and prepare destination address
	var addr syscall.Sockaddr
	var fd int
	var packet []byte

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if p.IsIPv6 {
		ip6 := p.NetworkLayer.(*layers.IPv6)

		newIP6 := &layers.IPv6{
			Version:    6,
			SrcIP:      ip6.DstIP, // Swap: our IP becomes source
			DstIP:      ip6.SrcIP, // Swap: server IP becomes destination
			NextHeader: layers.IPProtocolTCP,
			HopLimit:   s.InjectTTL,
		}
		newTCP.SetNetworkLayerForChecksum(newIP6)

		if err := gopacket.SerializeLayers(buffer, opts, newIP6, newTCP, gopacket.Payload(s.randomData[:])); err != nil {
			slog.Error("Failed to serialize packet", slog.Any("error", err))
			return
		}
		packet = buffer.Bytes()

		addr = &syscall.SockaddrInet6{
			Addr: [16]byte(ip6.SrcIP),
		}
		fd = s.rawSocketFD6

		slog.Debug("Injected third handshake packet",
			slog.String("src", fmt.Sprintf("%s:%d", newIP6.DstIP, newTCP.SrcPort)),
			slog.String("dst", fmt.Sprintf("%s:%d", newIP6.SrcIP, newTCP.DstPort)))
	} else {
		ip4 := p.NetworkLayer.(*layers.IPv4)
		newIP4 := &layers.IPv4{
			Version:  4,
			IHL:      5,
			SrcIP:    ip4.DstIP, // Swap: our IP becomes source
			DstIP:    ip4.SrcIP, // Swap: server IP becomes destination
			TTL:      s.InjectTTL,
			Protocol: layers.IPProtocolTCP,
		}
		newTCP.SetNetworkLayerForChecksum(newIP4)

		if err := gopacket.SerializeLayers(buffer, opts, newIP4, newTCP, gopacket.Payload(s.randomData[:])); err != nil {
			slog.Error("Failed to serialize packet", slog.Any("error", err))
			return
		}
		packet = buffer.Bytes()

		var ip4Bytes [4]byte
		copy(ip4Bytes[:], ip4.SrcIP.To4())
		addr = &syscall.SockaddrInet4{
			Addr: ip4Bytes,
		}
		fd = s.rawSocketFD4

		slog.Debug("Injected third handshake packet",
			slog.String("src", fmt.Sprintf("%s:%d", newIP4.DstIP, newTCP.SrcPort)),
			slog.String("dst", fmt.Sprintf("%s:%d", newIP4.SrcIP, newTCP.DstPort)))
	}

	if err := syscall.Sendto(fd, packet, 0, addr); err != nil {
		slog.Error("syscall.Sendto", slog.Any("error", err))
		return
	}
}

func (s *Server) checkTTL(p *base.Packet) bool {
	var ttl uint8
	var dis uint8

	if p.IsIPv6 {
		ip6 := p.NetworkLayer.(*layers.IPv6)
		ttl = ip6.HopLimit
	} else {
		ip4 := p.NetworkLayer.(*layers.IPv4)
		ttl = ip4.TTL
	}

	if ttl <= 64 {
		dis = 64 - ttl
	} else if 64 < ttl && ttl <= 128 {
		dis = 128 - ttl
	} else {
		dis = 255 - ttl
	}

	if s.cfg.TCPDesync.InjectTTL > dis {
		return false
	}
	return true
}
