package base

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Frame struct {
	A            *nfq.Attribute
	Ethernet     *layers.Ethernet
	NetworkLayer gopacket.NetworkLayer
	TCP          *layers.TCP
	SrcAddr      string
	DstAddr      string
	IsIPv6       bool
}

type FragmentConfig struct {
	Enable            bool
	FragmentSize      int
	OutOfOrder        bool
	MinFragments      int // 0 auto calculate
	FirstFragmentSize int // 0 means random 1-5 bytes
}

// NewFrame creates a Ethernet frame from the given nfqueue attribute.
func NewFrame(a *nfq.Attribute) (frame *Frame, err error) {
	frame = &Frame{
		A:        a,
		Ethernet: &layers.Ethernet{},
		TCP:      &layers.TCP{},
	}

	var decoded []gopacket.LayerType
	var ip4 layers.IPv4
	var ip6 layers.IPv6

	pktData := *a.Payload

	parser := gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		frame.Ethernet,
		&ip4,
		&ip6,
		frame.TCP,
	)

	if err = parser.DecodeLayers(pktData, &decoded); err != nil {
		return
	}

	// Determine IP version from Ethernet EthernetType
	for _, layerType := range decoded {
		switch layerType {
		case layers.LayerTypeIPv4:
			frame.NetworkLayer = &ip4
			frame.IsIPv6 = false
			frame.SrcAddr = fmt.Sprintf("%s:%d", ip4.SrcIP.String(), frame.TCP.SrcPort)
			frame.DstAddr = fmt.Sprintf("%s:%d", ip4.DstIP.String(), frame.TCP.DstPort)
		case layers.LayerTypeIPv6:
			frame.NetworkLayer = &ip6
			frame.IsIPv6 = true
			frame.SrcAddr = fmt.Sprintf("%s:%d", ip6.SrcIP.String(), frame.TCP.SrcPort)
			frame.DstAddr = fmt.Sprintf("%s:%d", ip6.DstIP.String(), frame.TCP.DstPort)
		}
	}

	return
}

// Serialize serializes the Frame back to a byte slice.
func (f *Frame) Serialize() ([]byte, error) {
	if err := f.TCP.SetNetworkLayerForChecksum(f.NetworkLayer); err != nil {
		return nil, err
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	var err error
	if f.IsIPv6 {
		ip6 := f.NetworkLayer.(*layers.IPv6)
		err = gopacket.SerializeLayers(buf, opts,
			f.Ethernet,
			ip6,
			f.TCP,
			gopacket.Payload(f.TCP.Payload),
		)
	} else {
		ip4 := f.NetworkLayer.(*layers.IPv4)
		err = gopacket.SerializeLayers(buf, opts,
			f.Ethernet,
			ip4,
			f.TCP,
			gopacket.Payload(f.TCP.Payload),
		)
	}
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (f *Frame) SerializeWithFragment() ([]byte, error) {
	fragmentedFrames, err := f.SerializeFragments(&FragmentConfig{
		Enable:            true,
		OutOfOrder:        false,
		MinFragments:      3,
		FirstFragmentSize: 10,
		FragmentSize:      256,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Serialized with fragmentation",
		slog.Int("Original Payload Size", len(f.TCP.Payload)),
		slog.Int("fragments", len(fragmentedFrames)),
		slog.String("SrcAddr", f.SrcAddr),
		slog.String("DstAddr", f.DstAddr))

	combined := []byte{}
	for _, frag := range fragmentedFrames {
		combined = append(combined, frag...)
	}
	return combined, nil
}

func (f *Frame) SerializeFragments(cfg *FragmentConfig) ([][]byte, error) {
	if cfg == nil || !cfg.Enable {
		data, err := f.Serialize()
		if err != nil {
			return nil, err
		}
		return [][]byte{data}, nil
	}

	payload := f.TCP.Payload
	if len(payload) == 0 || f.TCP.FIN {
		data, err := f.Serialize()
		if err != nil {
			return nil, err
		}
		return [][]byte{data}, nil
	}

	fragmentSize := cfg.FragmentSize
	var numFragments int

	if fragmentSize <= 0 {
		// fragmentSize not specified, calculate from MinFragments
		if cfg.MinFragments > 0 {
			numFragments = cfg.MinFragments
			fragmentSize = (len(payload) + numFragments - 1) / numFragments
		} else {
			// default to 2 fragments if neither is specified
			numFragments = 2
			fragmentSize = (len(payload) + 1) / 2
		}
	} else {
		numFragments = (len(payload) + fragmentSize - 1) / fragmentSize
		if cfg.MinFragments > 0 && numFragments < cfg.MinFragments {
			numFragments = cfg.MinFragments
			fragmentSize = (len(payload) + numFragments - 1) / numFragments
		}
	}

	if numFragments < 2 && len(payload) >= 2 {
		numFragments = 2
		fragmentSize = (len(payload) + 1) / 2
	}

	type fragment struct {
		offset int
		length int
		seq    uint32
	}

	fragments := make([]fragment, 0, numFragments)
	offset := 0
	baseSeq := f.TCP.Seq

	// first fragment
	firstSize := cfg.FirstFragmentSize
	if firstSize <= 0 {
		// random 1-5 bytes
		n, err := rand.Int(rand.Reader, big.NewInt(5))
		if err != nil {
			firstSize = 3
		} else {
			firstSize = int(n.Int64()) + 1
		}
	}
	if firstSize > len(payload) {
		firstSize = len(payload)
	}

	fragments = append(fragments, fragment{
		offset: 0,
		length: firstSize,
		seq:    baseSeq,
	})
	offset = firstSize

	// remaining fragments
	for offset < len(payload) {
		length := fragmentSize
		if offset+length > len(payload) {
			length = len(payload) - offset
		}
		fragments = append(fragments, fragment{
			offset: offset,
			length: length,
			seq:    baseSeq + uint32(offset),
		})
		offset += length
	}

	// Fisher-Yates
	if cfg.OutOfOrder && len(fragments) > 1 {
		for i := len(fragments) - 1; i > 0; i-- {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
			if err != nil {
				continue
			}
			j := int(n.Int64())
			fragments[i], fragments[j] = fragments[j], fragments[i]
		}
	}

	// serialize fragments
	packets := make([][]byte, 0, len(fragments))
	for i, frag := range fragments {
		data, err := f.serializeFragment(payload[frag.offset:frag.offset+frag.length], frag.seq)
		if err != nil {
			return nil, fmt.Errorf("serialize fragment at offset %d: %w", frag.offset, err)
		}
		slog.Info("Serialized fragment",
			slog.Int("Fragment Index", i),
			slog.Int("Fragment Size", frag.length),
			slog.String("SrcAddr", f.SrcAddr),
			slog.String("DstAddr", f.DstAddr))
		packets = append(packets, data)
	}

	return packets, nil
}

// serializeFragment serializes a single tcp fragment
// return ethernet frame
func (f *Frame) serializeFragment(fragmentPayload []byte, seq uint32) ([]byte, error) {
	// Create a copy of TCP layer with modified seq and payload
	tcpCopy := *f.TCP
	tcpCopy.Seq = seq
	tcpCopy.Payload = fragmentPayload

	if err := tcpCopy.SetNetworkLayerForChecksum(f.NetworkLayer); err != nil {
		return nil, err
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	var err error
	if f.IsIPv6 {
		ip6 := f.NetworkLayer.(*layers.IPv6)
		err = gopacket.SerializeLayers(buf, opts,
			f.Ethernet,
			ip6,
			&tcpCopy,
			gopacket.Payload(fragmentPayload),
		)
	} else {
		ip4 := f.NetworkLayer.(*layers.IPv4)
		err = gopacket.SerializeLayers(buf, opts,
			f.Ethernet,
			ip4,
			&tcpCopy,
			gopacket.Payload(fragmentPayload),
		)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
