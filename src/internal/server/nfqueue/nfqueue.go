package nfqueue

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	nfq "github.com/florianl/go-nfqueue/v2"
	"github.com/mdlayher/netlink"
	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	cfg                 *config.Config
	rw                  *rewrite.Rewriter
	nf                  *nfq.Nfqueue
	queueNum            uint16
	maxQueueLen         uint32
	maxPacketLen        uint32
	numWorkers          int
	workers             []chan *nfq.Attribute
	wg                  sync.WaitGroup
	SniffMarkRangeLower uint32
	SniffMarkRangeUpper uint32
	HTTPMark            uint32
	NotHTTPMark         uint32
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}

	return &Server{
		cfg:                 cfg,
		rw:                  rw,
		queueNum:            10201,
		maxQueueLen:         2000,
		numWorkers:          numWorkers,
		SniffMarkRangeLower: 10201,
		SniffMarkRangeUpper: 10216,
		NotHTTPMark:         201,
		HTTPMark:            202,
	}
}

// worker processes packets from its assigned channel
func (s *Server) worker(ctx context.Context, workerID int, aChan <-chan *nfq.Attribute) {
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
			if s.cfg.RewriteMode == config.RewriteModeDirect {
				_ = s.nf.SetVerdict(*a.PacketID, nfq.NfAccept)
				continue
			} else {
				s.handlePacket(a)
			}
		}
	}
}

func (s *Server) computeWorkerIndex(a *nfq.Attribute) int {
	var flowID uint32
	if a.Ct != nil {
		flowID = ctIDFromCtBytes(*a.Ct)
	} else {
		// Compute flow hash to determine which worker should handle this packet
		flowID = computeFlowHash(*a.Payload)
	}
	workerIdx := int(flowID % uint32(s.numWorkers))
	return workerIdx
}

func (s *Server) SendVerdict(packet *IPPacket, result *rewrite.RewriteResult) {
	nf := s.nf
	id := *packet.a.PacketID
	setMark, nextMark := s.getNextMark(packet, result)

	var newPacket []byte
	var err error

	if result.Modified {
		newPacket, err = serializeIPPacket(packet.NetworkLayer, packet.TCP, result.NewPayload, packet.IsIPv6)
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

// handlePacket processes a single NFQUEUE packet
func (s *Server) handlePacket(a *nfq.Attribute) {
	nf := s.nf
	if ok, verdict := packetAttributeSanityCheck(a); !ok {
		if a.PacketID != nil {
			_ = nf.SetVerdict(*a.PacketID, verdict)
		}
		return
	}
	packet, err := decodeIPPacket(a)
	if err != nil {
		_ = nf.SetVerdict(*a.PacketID, nfq.NfAccept)
		return
	}
	if s.rw.Cache.Contains(packet.DstAddr) {
		s.SendVerdict(packet, &rewrite.RewriteResult{Modified: false, InCache: true})
		log.LogDebugWithAddr(packet.SrcAddr, packet.DstAddr, "Destination in cache, skipping User-Agent rewrite")
		return
	}
	result := s.rw.RewriteTCP(packet.TCP, packet.SrcAddr, packet.DstAddr)
	s.SendVerdict(packet, result)
}

func (s *Server) getNextMark(packet *IPPacket, result *rewrite.RewriteResult) (setMark bool, mark uint32) {
	mark, found := getConnMark(packet.a)
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

func (s *Server) Start() (err error) {
	logrus.Infof("Starting NFQUEUE mode on queue %d with %d workers...", s.queueNum, s.numWorkers)

	config := nfq.Config{
		NfQueue:      s.queueNum,
		MaxQueueLen:  s.maxQueueLen,
		MaxPacketLen: s.maxPacketLen,
		Copymode:     nfq.NfQnlCopyPacket,
		Flags:        nfq.NfQaCfgFlagConntrack,
	}

	nf, err := nfq.Open(&config)
	if err != nil {
		return fmt.Errorf("nfq.Open: %w", err)
	}
	defer nf.Close()
	s.nf = nf

	// Ignore ENOBUFS to prevent queue drop logs
	if err := nf.SetOption(netlink.NoENOBUFS, true); err != nil {
		return fmt.Errorf("nf.SetOption: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize worker channels and start worker goroutines
	s.workers = make([]chan *nfq.Attribute, s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		s.workers[i] = make(chan *nfq.Attribute, 2000)
		s.wg.Add(1)
		go s.worker(ctx, i, s.workers[i])
	}

	// Register callback function
	err = nf.RegisterWithErrorFunc(ctx,
		func(a nfq.Attribute) int {
			select {
			case s.workers[s.computeWorkerIndex(&a)] <- &a:
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
			return 0
		},
	)
	if err != nil {
		// Close all worker channels
		for i := 0; i < s.numWorkers; i++ {
			close(s.workers[i])
		}
		s.wg.Wait()
		return fmt.Errorf("failed to register nfqueue handler: %w", err)
	}

	logrus.Info("NFQUEUE handler registered, listening for packets")

	go statistics.StartRecorder()

	<-ctx.Done()

	// Cleanup: close all worker channels and wait for workers to finish
	for i := 0; i < s.numWorkers; i++ {
		close(s.workers[i])
	}
	s.wg.Wait()

	return nil
}
