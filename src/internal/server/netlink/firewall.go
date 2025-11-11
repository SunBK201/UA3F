package netlink

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
)

const (
	NFT = "nft"
	IPT = "ipt"
)

func (s *Server) setupFirewall() error {
	s.cleanupFirewall()
	backend := s.detectFirewallBackend()
	switch backend {
	case NFT:
		return s.nftSetup()
	case IPT:
		return s.iptSetup()
	default:
		return fmt.Errorf("unsupported or no firewall backend: %s", backend)
	}
}

func (s *Server) cleanupFirewall() error {
	s.nftCleanup()
	s.iptCleanup()
	return nil
}

func (s *Server) detectFirewallBackend() string {
	// Check if opkg is available (OpenWrt environment)
	if isCommandAvailable("opkg") {
		switch s.cfg.ServerMode {
		case config.ServerModeTProxy:
			// Check if kmod-nft-tproxy is installed
			if isOpkgPackageInstalled("kmod-nft-tproxy") && isCommandAvailable("nft") {
				logrus.Info("Detected nftables backend (kmod-nft-tproxy installed)")
				return NFT
			}
			logrus.Info("Detected iptables backend (kmod-nft-tproxy not installed)")
			return IPT
		case config.ServerModeNFQueue:
			// Check if kmod-nft-queue is installed
			if isOpkgPackageInstalled("kmod-nft-queue") && isCommandAvailable("nft") {
				logrus.Info("Detected nftables backend (kmod-nft-queue installed)")
				return NFT
			}
			logrus.Info("Detected iptables backend (kmod-nft-queue not installed)")
			return IPT
		}
	}

	// Check if nft command is available
	if isCommandAvailable("nft") {
		logrus.Info("Detected nftables backend (nft command available)")
		return NFT
	}

	// Check if iptables command is available
	if isCommandAvailable("iptables") {
		logrus.Info("Detected iptables backend (iptables command available)")
		return IPT
	}

	// No backend detected
	logrus.Warn("No firewall backend detected")
	return ""
}
