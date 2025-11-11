package netfilter

import (
	"fmt"
	"os/exec"

	"github.com/gonetx/ipset"
	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"sigs.k8s.io/knftables"
)

const (
	NFT = "nft"
	IPT = "ipt"
)

const (
	LANSET       = "UA3F_LAN"
	SKIP_PORTS   = "22,51080,51090"
	FAKEIP_RANGE = "198.18.0.0/16,198.18.0.1/15,28.0.0.1/8"
	HELPER_QUEUE = 10301
)

var LAN_CIDRS = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.1.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
}

var RuleIgnoreReply = []string{
	"-m", "conntrack",
	"--ctdir", "REPLY",
	"-j", "RETURN",
}

var RuleIgnoreLAN = []string{
	"-m", "set",
	"--match-set", LANSET, "dst",
	"-j", "RETURN",
}

var RuleIgnorePorts = []string{
	"-p", "tcp",
	"-m", "multiport",
	"--dports", SKIP_PORTS,
	"-j", "RETURN",
}

type Firewall struct {
	NftSetup   func() error
	NftCleanup func() error
	IptSetup   func() error
	IptCleanup func() error
}

func (f *Firewall) Setup(cfg *config.Config) (err error) {
	f.Cleanup()
	backend := detectFirewallBackend(cfg)
	switch backend {
	case NFT:
		err = f.NftSetup()
	case IPT:
		err = f.IptSetup()
	default:
		err = fmt.Errorf("unsupported or no firewall backend: %s", backend)
	}
	if err != nil {
		f.Cleanup()
	}
	return err
}

func (f *Firewall) Cleanup() error {
	f.NftCleanup()
	f.IptCleanup()
	return nil
}

func (f *Firewall) NftSetLanIP(tx *knftables.Transaction, table *knftables.Table) {
	ipset := &knftables.Set{
		Name:   LANSET,
		Table:  table.Name,
		Family: table.Family,
		Type:   "ipv4_addr",
		Flags: []knftables.SetFlag{
			knftables.IntervalFlag,
		},
		AutoMerge: knftables.PtrTo(true),
	}
	tx.Add(ipset)

	for _, cidr := range LAN_CIDRS {
		iplan := &knftables.Element{
			Table:  table.Name,
			Family: table.Family,
			Set:    ipset.Name,
			Key:    []string{cidr},
		}
		tx.Add(iplan)
	}
}

func (f *Firewall) IptSetLanIP() error {
	if err := ipset.Check(); err != nil {
		return err
	}
	set, err := ipset.New(
		LANSET,
		ipset.HashNet,
		ipset.Exist(false),
		ipset.Family(ipset.Inet),
	)
	if err != nil {
		return err
	}

	for _, cidr := range LAN_CIDRS {
		err := set.Add(cidr)
		if err != nil {
			return err
		}
	}
	set.Flush()
	return nil
}

func (f *Firewall) IptDeleteLanIP() error {
	return ipset.Destroy(LANSET)
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

// isCommandAvailable checks if a command is available in the system
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// isOpkgPackageInstalled checks if a package is installed via opkg
func isOpkgPackageInstalled(pkg string) bool {
	cmd := exec.Command("opkg", "list-installed", pkg)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}
