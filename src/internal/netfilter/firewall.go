package netfilter

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
)

const (
	NFT = "nft"
	IPT = "ipt"
)

type Firewall struct {
	NftSetup   func() error
	NftCleanup func() error
	IptSetup   func() error
	IptCleanup func() error
}

func (f *Firewall) Setup(cfg *config.Config) error {
	f.Cleanup()
	backend := detectFirewallBackend(cfg)
	switch backend {
	case NFT:
		return f.NftSetup()
	case IPT:
		return f.IptSetup()
	default:
		return fmt.Errorf("unsupported or no firewall backend: %s", backend)
	}
}

func (f *Firewall) Cleanup() error {
	f.NftCleanup()
	f.IptCleanup()
	return nil
}

func detectFirewallBackend(cfg *config.Config) string {
	// Check if opkg is available (OpenWrt environment)
	if isCommandAvailable("opkg") {
		switch cfg.ServerMode {
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

// IsCommandAvailable checks if a command is available in the system
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsOpkgPackageInstalled checks if a package is installed via opkg
func isOpkgPackageInstalled(pkg string) bool {
	cmd := exec.Command("opkg", "list-installed", pkg)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}
