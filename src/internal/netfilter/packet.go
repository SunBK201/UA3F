package netfilter

import (
	"encoding/binary"
	"fmt"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/netlink"
	"github.com/sirupsen/logrus"
)

type Packet struct {
	A            *nfq.Attribute
	NetworkLayer gopacket.NetworkLayer
	TCP          *layers.TCP
	SrcAddr      string
	DstAddr      string
	IsIPv6       bool
}

func NewPacket(a *nfq.Attribute) (packet *Packet, err error) {
	packet = &Packet{
		A:   a,
		TCP: &layers.TCP{},
	}

	var decoded []gopacket.LayerType
	var layerType gopacket.LayerType
	var ipLayer gopacket.DecodingLayer

	pktData := *a.Payload
	version := (pktData[0] >> 4) & 0xF
	packet.IsIPv6 = version == 6

	if packet.IsIPv6 {
		ip6 := &layers.IPv6{}
		layerType = layers.LayerTypeIPv6
		ipLayer = ip6
		packet.NetworkLayer = ip6
	} else {
		ip4 := &layers.IPv4{}
		layerType = layers.LayerTypeIPv4
		ipLayer = ip4
		packet.NetworkLayer = ip4
	}

	parser := gopacket.NewDecodingLayerParser(layerType, ipLayer, packet.TCP)
	parser.IgnoreUnsupported = true

	if err = parser.DecodeLayers(pktData, &decoded); err != nil {
		return
	}

	if packet.IsIPv6 {
		ip6 := packet.NetworkLayer.(*layers.IPv6)
		packet.SrcAddr = fmt.Sprintf("%s:%d", ip6.SrcIP.String(), packet.TCP.SrcPort)
		packet.DstAddr = fmt.Sprintf("%s:%d", ip6.DstIP.String(), packet.TCP.DstPort)
	} else {
		ip4 := packet.NetworkLayer.(*layers.IPv4)
		packet.SrcAddr = fmt.Sprintf("%s:%d", ip4.SrcIP.String(), packet.TCP.SrcPort)
		packet.DstAddr = fmt.Sprintf("%s:%d", ip4.DstIP.String(), packet.TCP.DstPort)
	}
	return
}

func (p *Packet) Serialize() ([]byte, error) {
	networkLayer := p.NetworkLayer
	tcp := p.TCP
	isIPv6 := p.IsIPv6
	newPayload := tcp.Payload

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

func (p *Packet) GetCtMark() (uint32, bool) {
	if p.A.Ct == nil || len(*p.A.Ct) == 0 {
		return 0, false
	}

	attrs, err := netlink.UnmarshalAttributes(*p.A.Ct)
	if err != nil {
		logrus.Errorf("netlink.UnmarshalAttributes: %s", err.Error())
		return 0, false
	}

	for _, at := range attrs {
		if at.Type == 8 && len(at.Data) >= 4 { // CTA_MARK
			return binary.BigEndian.Uint32(at.Data[:4]), true
		}
	}

	return 0, false
}

func (p *Packet) GetCtID() (uint32, bool) {
	if p.A.Ct == nil || len(*p.A.Ct) == 0 {
		return 0, false
	}

	attrs, err := netlink.UnmarshalAttributes(*p.A.Ct)
	if err != nil {
		logrus.Errorf("netlink.UnmarshalAttributes: %s", err.Error())
		return 0, false
	}
	for _, at := range attrs {
		if at.Type == 12 { // CTA_ID
			return binary.BigEndian.Uint32(at.Data), true
		}
	}

	return 0, false
}
