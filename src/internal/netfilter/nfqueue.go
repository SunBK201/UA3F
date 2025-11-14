package netfilter

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"runtime"
	"strings"
	"sync"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/mdlayher/netlink"
	"github.com/sirupsen/logrus"
)

type NfqHandler func(a *Packet)

type NfqueueServer struct {
	QueueNum      uint16
	MaxQueueLen   uint32
	MaxPacketLen  uint32
	HandlePacket  NfqHandler
	NumWorkers    int
	WorkerChanLen int
	attrChans     []chan *nfq.Attribute
	wg            sync.WaitGroup
	Nf            *nfq.Nfqueue
}

func (s *NfqueueServer) Start() error {
	if s.QueueNum == 0 {
		return fmt.Errorf("NfqueueServer.QueueNum is 0")
	}
	if s.HandlePacket == nil {
		return fmt.Errorf("NfqueueServer.Handler is nil")
	}
	if s.MaxQueueLen <= 0 {
		s.MaxQueueLen = 2000
	}
	if s.MaxPacketLen <= 0 {
		s.MaxPacketLen = 1600
	}
	if s.NumWorkers <= 0 {
		s.NumWorkers = runtime.NumCPU()
		if s.NumWorkers < 2 {
			s.NumWorkers = 2
		}
	}
	if s.WorkerChanLen <= 0 {
		s.WorkerChanLen = 2000
	}
	config := nfq.Config{
		NfQueue:      s.QueueNum,
		MaxQueueLen:  s.MaxQueueLen,
		MaxPacketLen: s.MaxPacketLen,
		Copymode:     nfq.NfQnlCopyPacket,
		Flags:        nfq.NfQaCfgFlagConntrack,
	}

	nf, err := nfq.Open(&config)
	if err != nil {
		return fmt.Errorf("nfq.Open: %w", err)
	}
	defer func() {
		if cerr := nf.Close(); cerr != nil {
			logrus.Errorf("nf.Close: %v", cerr)
		}
	}()
	s.Nf = nf

	// Ignore ENOBUFS to prevent queue drop logs
	// if err := nf.SetOption(netlink.NoENOBUFS, true); err != nil {
	//	return fmt.Errorf("nf.SetOption: %w", err)
	// }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize worker channels and start worker goroutines
	s.attrChans = make([]chan *nfq.Attribute, s.NumWorkers)
	for i := 0; i < s.NumWorkers; i++ {
		s.attrChans[i] = make(chan *nfq.Attribute, s.WorkerChanLen)
		s.wg.Add(1)
		go s.worker(ctx, i, s.attrChans[i])
	}

	// Register callback function
	err = nf.RegisterWithErrorFunc(ctx,
		func(a nfq.Attribute) int {
			select {
			case s.attrChans[s.computeWorkerIndex(&a)] <- &a:
			default:
				// If worker channel is full, accept the packet to avoid blocking
				logrus.Warn("Worker channel full, accepting packet without processing")
				if a.PacketID != nil {
					_ = nf.SetVerdict(*a.PacketID, nfq.NfAccept)
				}
			}
			return 0
		},
		func(e error) int {
			logrus.Errorf("Error in nfqueue handler: %v", e)
			if strings.Contains(e.Error(), "no buffer space available") {
				logrus.Warnf("Consider increasing the read buffer size to prevent packet drops")
				err = nf.Con.SetReadBuffer(1024 * 1024 * 5)
				if err != nil {
					logrus.Errorf("nf.Con.SetReadBuffer: %v", err)
				}
			}
			return 0
		},
	)
	if err != nil {
		// Close all worker channels
		for i := 0; i < s.NumWorkers; i++ {
			close(s.attrChans[i])
		}
		s.wg.Wait()
		return fmt.Errorf("failed to register nfqueue handler: %w", err)
	}

	// Wait until context is done
	<-ctx.Done()
	// Cleanup: close all worker channels and wait for workers to finish
	for i := 0; i < s.NumWorkers; i++ {
		close(s.attrChans[i])
	}
	s.wg.Wait()
	return nil
}

// worker processes packets from its assigned channel
func (s *NfqueueServer) worker(ctx context.Context, workerID int, aChan <-chan *nfq.Attribute) {
	defer s.wg.Done()
	logrus.Debugf("Worker %d started", workerID)
	for {
		select {
		case <-ctx.Done():
			logrus.Debugf("Worker %d stopping", workerID)
			return
		case a, ok := <-aChan:
			if !ok {
				logrus.Debugf("Worker %d channel closed", workerID)
				return
			}
			if ok, verdict := AttributeSanityCheck(a); !ok {
				if a.PacketID != nil {
					_ = s.Nf.SetVerdict(*a.PacketID, verdict)
				}
				return
			}
			packet, err := NewPacket(a)
			if err != nil {
				logrus.Errorf("Worker %d: NewPacket error: %v", workerID, err)
				if a.PacketID != nil {
					_ = s.Nf.SetVerdict(*a.PacketID, nfq.NfAccept)
				}
				continue
			}
			s.HandlePacket(packet)
		}
	}
}

func (s *NfqueueServer) computeWorkerIndex(a *nfq.Attribute) int {
	var flowID uint32
	if a.Ct != nil {
		flowID = ctIDFromCtBytes(*a.Ct)
	} else {
		// Compute flow hash to determine which worker should handle this packet
		flowID = computeFlowHash(*a.Payload)
	}
	workerIdx := int(flowID % uint32(s.NumWorkers))
	return workerIdx
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

func AttributeSanityCheck(a *nfq.Attribute) (ok bool, verdict int) {
	if a.PacketID == nil {
		return false, -1
	}
	if a.Payload == nil || len(*a.Payload) < 20 {
		return false, nfq.NfAccept
	}
	return true, 0
}
