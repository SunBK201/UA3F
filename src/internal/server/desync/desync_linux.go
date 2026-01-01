//go:build linux

package desync

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"syscall"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"github.com/sunbk201/ua3f/internal/server/base"
	"sigs.k8s.io/knftables"
)

type Server struct {
	netfilter.Firewall
	cfg              *config.Config
	ReorderNfqServer *base.NfqueueServer
	DesyncPorts      []uint16

	ReorderByte    uint32
	ReorderPackets uint32

	InjectNfqServer *base.NfqueueServer
	randomData      [64]byte
	InjectTTL       uint8
	rawSocketFD4    int
	rawSocketFD6    int
	InjectMark      int
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		ReorderNfqServer: &base.NfqueueServer{
			QueueNum: netfilter.DESYNC_REORDER_QUEUE,
		},
		InjectNfqServer: &base.NfqueueServer{
			QueueNum: netfilter.DESYNC_INJECT_QUEUE,
		},
		ReorderByte:    1500,
		ReorderPackets: 2 + 3*2,
		InjectTTL:      3,
		InjectMark:     base.SO_INJECT_MARK,
	}
	s.ReorderNfqServer.HandlePacket = s.ReorderPacket
	s.InjectNfqServer.HandlePacket = s.InjectPacket
	s.Firewall = netfilter.Firewall{
		Nftable: &knftables.Table{
			Name:   "UA3F_DESYNC",
			Family: knftables.InetFamily,
		},
		NftSetup:   s.nftSetup,
		NftCleanup: s.nftCleanup,
		IptSetup:   s.iptSetup,
		IptCleanup: s.iptCleanup,
	}
	if s.cfg.Desync.ReorderBytes > 0 {
		s.ReorderByte = s.cfg.Desync.ReorderBytes
	}
	if s.cfg.Desync.ReorderPackets > 0 {
		s.ReorderPackets = s.cfg.Desync.ReorderPackets
	}
	if s.cfg.Desync.InjectTTL > 0 {
		s.InjectTTL = s.cfg.Desync.InjectTTL
	}

	if s.cfg.Desync.DesyncPorts != "" {
		ports, err := parsePorts(s.cfg.Desync.DesyncPorts)
		if err != nil {
			slog.Error("parsePorts", slog.Any("error", err))
		} else {
			s.DesyncPorts = ports
		}
	}
	return s
}

func (s *Server) Start() (err error) {
	err = s.Firewall.Setup(s.cfg)
	if err != nil {
		slog.Error("s.Firewall.Setup", slog.Any("error", err))
		return err
	}

	if s.cfg.Desync.Reorder {
		err = s.ReorderNfqServer.Start()
		if err != nil {
			return err
		}
	}

	if s.cfg.Desync.Inject {
		s.rawSocketFD4, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
		if err != nil {
			return err
		}
		err = syscall.SetsockoptInt(s.rawSocketFD4, syscall.SOL_SOCKET, syscall.SO_MARK, s.InjectMark)
		if err != nil {
			return err
		}
		err = syscall.SetsockoptInt(s.rawSocketFD4, syscall.SOL_SOCKET, syscall.SO_PRIORITY, 7)
		if err != nil {
			slog.Error("syscall.SetsockoptInt SO_PRIORITY", slog.Any("error", err))
		}
		err = syscall.SetsockoptInt(s.rawSocketFD4, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 128)
		if err != nil {
			slog.Error("syscall.SetsockoptInt SO_RCVBUF", slog.Any("error", err))
		}
		s.rawSocketFD6, _ = syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
		if s.rawSocketFD6 > 0 {
			err = syscall.SetsockoptInt(s.rawSocketFD6, syscall.SOL_SOCKET, syscall.SO_MARK, s.InjectMark)
			if err != nil {
				return err
			}
			err = syscall.SetsockoptInt(s.rawSocketFD6, syscall.SOL_SOCKET, syscall.SO_PRIORITY, 7)
			if err != nil {
				slog.Error("syscall.SetsockoptInt SO_PRIORITY", slog.Any("error", err))
			}
			err = syscall.SetsockoptInt(s.rawSocketFD6, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 128)
			if err != nil {
				slog.Error("syscall.SetsockoptInt SO_RCVBUF", slog.Any("error", err))
			}
		}
		if _, err := rand.Read(s.randomData[:]); err != nil {
			slog.Error("rand.Read", slog.Any("error", err))
		}
		err = s.InjectNfqServer.Start()
		if err != nil {
			return err
		}
	}

	slog.Info("TCP Desync server started", slog.Int("reorder_bytes", int(s.ReorderByte)), slog.Int("reorder_packets", int(s.ReorderPackets)), slog.Int("inject_ttl", int(s.InjectTTL)))

	return
}

func (s *Server) Close() error {
	err := s.Firewall.Cleanup()
	if s.cfg.Desync.Reorder {
		s.ReorderNfqServer.Close()
	}
	if s.cfg.Desync.Inject {
		s.InjectNfqServer.Close()
		syscall.Close(s.rawSocketFD4)
		syscall.Close(s.rawSocketFD6)
	}
	return err
}

func parsePorts(s string) ([]uint16, error) {
	parts := strings.Split(s, ",")
	ports := make([]uint16, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		v, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}

		ports = append(ports, uint16(v))
	}

	return ports, nil
}
