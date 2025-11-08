package nfqueue

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/netlink"
	"github.com/sirupsen/logrus"
)

type IPPacket struct {
	a            *nfq.Attribute
	NetworkLayer gopacket.NetworkLayer
	TCP          *layers.TCP
	SrcAddr      string
	DstAddr      string
	IsIPv6       bool
}

func packetAttributeSanityCheck(a *nfq.Attribute) (ok bool, verdict int) {
	if a.PacketID == nil {
		return false, -1
	}
	if a.Payload == nil || len(*a.Payload) < 40 {
		return false, nfq.NfAccept
	}
	return true, 0
}

// computeFlowHash computes a hash value based on TCP 4-tuple to ensure packets
// from the same TCP stream are handled by the same worker goroutine
func computeFlowHash(pktData []byte) uint32 {
	version := (pktData[0] >> 4) & 0xF

	h := fnv.New32a()

	switch version {
	case 4:
		// IPv4: IP header is at least 20 bytes
		if len(pktData) < 20 {
			return 0
		}

		// Source IP (bytes 12-15) and Dest IP (bytes 16-19)
		h.Write(pktData[12:20])

		// Check if it's TCP (protocol 6)
		protocol := pktData[9]
		if protocol == 6 {
			// IHL (IP Header Length) is in the lower 4 bits of byte 0
			ihl := (pktData[0] & 0x0F) * 4
			if len(pktData) >= int(ihl)+4 {
				// TCP source port and dest port (first 4 bytes of TCP header)
				h.Write(pktData[ihl : ihl+4])
			}
		}

	case 6:
		// IPv6: IP header is at least 40 bytes
		if len(pktData) < 40 {
			return 0
		}

		// Source IP (bytes 8-23) and Dest IP (bytes 24-39)
		h.Write(pktData[8:40])

		// Check if it's TCP (next header 6)
		nextHeader := pktData[6]
		if nextHeader == 6 && len(pktData) >= 44 {
			// TCP source port and dest port (first 4 bytes of TCP header at offset 40)
			h.Write(pktData[40:44])
		}
	}

	return h.Sum32()
}

func ctIDFromCtBytes(ct []byte) uint32 {
	ctAttrs, err := netlink.UnmarshalAttributes(ct)
	if err != nil {
		return 0
	}
	for _, attr := range ctAttrs {
		if attr.Type == 12 { // CTA_ID
			return binary.BigEndian.Uint32(attr.Data)
		}
	}
	return 0
}

// decodeIPPacket decodes IP packet and extracts TCP layer
func decodeIPPacket(a *nfq.Attribute) (ipPacket *IPPacket, err error) {
	ipPacket = &IPPacket{
		a: a,
	}

	var decoded []gopacket.LayerType
	var layerType gopacket.LayerType
	var ipLayer gopacket.DecodingLayer

	ipPacket.TCP = &layers.TCP{}

	pktData := *a.Payload
	version := (pktData[0] >> 4) & 0xF
	ipPacket.IsIPv6 = version == 6

	if ipPacket.IsIPv6 {
		ip6 := &layers.IPv6{}
		layerType = layers.LayerTypeIPv6
		ipLayer = ip6
		ipPacket.NetworkLayer = ip6
	} else {
		ip4 := &layers.IPv4{}
		layerType = layers.LayerTypeIPv4
		ipLayer = ip4
		ipPacket.NetworkLayer = ip4
	}

	parser := gopacket.NewDecodingLayerParser(layerType, ipLayer, ipPacket.TCP)
	parser.IgnoreUnsupported = true

	if err = parser.DecodeLayers(pktData, &decoded); err != nil {
		return
	}

	if ipPacket.IsIPv6 {
		ip6 := ipPacket.NetworkLayer.(*layers.IPv6)
		ipPacket.SrcAddr = fmt.Sprintf("%s:%d", ip6.SrcIP.String(), ipPacket.TCP.SrcPort)
		ipPacket.DstAddr = fmt.Sprintf("%s:%d", ip6.DstIP.String(), ipPacket.TCP.DstPort)
	} else {
		ip4 := ipPacket.NetworkLayer.(*layers.IPv4)
		ipPacket.SrcAddr = fmt.Sprintf("%s:%d", ip4.SrcIP.String(), ipPacket.TCP.SrcPort)
		ipPacket.DstAddr = fmt.Sprintf("%s:%d", ip4.DstIP.String(), ipPacket.TCP.DstPort)
	}
	return
}

// serializeIPPacket serializes IP packet with modified TCP payload
func serializeIPPacket(networkLayer gopacket.NetworkLayer, tcp *layers.TCP, newPayload []byte, isIPv6 bool) ([]byte, error) {
	buffer := gopacket.NewSerializeBuffer()
	serOpts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	tcp.Checksum = 0
	tcp.Payload = nil
	tcp.SetNetworkLayerForChecksum(networkLayer)

	var err error
	if isIPv6 {
		ip6 := networkLayer.(*layers.IPv6)
		ip6.NextHeader = layers.IPProtocolTCP
		err = gopacket.SerializeLayers(buffer, serOpts, ip6, tcp, gopacket.Payload(newPayload))
	} else {
		ip4 := networkLayer.(*layers.IPv4)
		ip4.Checksum = 0
		err = gopacket.SerializeLayers(buffer, serOpts, ip4, tcp, gopacket.Payload(newPayload))
	}

	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func getConnMark(a *nfq.Attribute) (uint32, bool) {
	if a.Ct == nil || len(*a.Ct) == 0 {
		return 0, false
	}

	attrs, err := netlink.UnmarshalAttributes(*a.Ct)
	if err != nil {
		logrus.Errorf("netlink.UnmarshalAttributes: %s", err.Error())
		return 0, false
	}

	for _, at := range attrs {
		if at.Type == 8 && len(at.Data) >= 4 {
			return binary.BigEndian.Uint32(at.Data[:4]), true
		}
	}

	return 0, false
}
