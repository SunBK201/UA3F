package netfilter

import (
	"fmt"
	"os/exec"
	"os/user"

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
	SO_MARK      = 0xc9
)

const (
	OC = "openclash"
	SC = "shellcrash"
)

var SIDECAR = OC
var SKIP_GIDS = "453"

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

var (
	IptRuleIgnoreBrLAN = []string{
		"!", "-i", "br-lan",
		"-j", "RETURN",
	}
	IptRuleIgnoreReply = []string{
		"-m", "conntrack",
		"--ctdir", "REPLY",
		"-j", "RETURN",
	}
	IptRuleIgnoreLAN = []string{
		"-m", "set",
		"--match-set", LANSET, "dst",
		"-j", "RETURN",
	}
	IptRuleIgnorePorts = []string{
		"-p", "tcp",
		"-m", "multiport",
		"--dports", SKIP_PORTS,
		"-j", "RETURN",
	}
)
var (
	NftRuleIgnoreNotTCP = knftables.Concat(
		"meta l4proto != tcp",
		"return",
	)
	NftRuleIgnoreBrLAN = knftables.Concat(
		"iifname != \"br-lan\"",
		"return",
	)
	NftRuleIgnoreReply = knftables.Concat(
		"ct direction reply",
		"return",
	)
	NftRuleIgnoreLAN = knftables.Concat(
		fmt.Sprintf("ip daddr @%s", LANSET),
		"return",
	)
	NftRuleIgnorePorts = knftables.Concat(
		fmt.Sprintf("tcp dport { %s }", SKIP_PORTS),
		"return",
	)
	NftRuleIgnoreFakeIP = knftables.Concat(
		fmt.Sprintf("ip daddr { %s }", FAKEIP_RANGE),
		"return",
	)
)

func init() {
	initSkipGids()
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

func (f *Firewall) DumpNFTables() {
	cmd := exec.Command("nft", "--handle", "list", "ruleset")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	logrus.Infof("nftables ruleset:\n%s", string(output))
}

func (f *Firewall) DumpIPTables() {
	var tables = []string{"filter", "nat", "mangle", "raw"}
	for _, table := range tables {
		cmd := exec.Command("iptables", "-t", table, "-L", "-v", "-n", "--line-numbers")
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		logrus.Infof("iptables table(%s):\n%s", table, string(output))
	}
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

func (f *Firewall) AddTproxyRoute(fwmark, routeTable string) error {
	sysctlCmds := [][]string{
		{"-w", "net.bridge.bridge-nf-call-iptables=0"},
		{"-w", "net.bridge.bridge-nf-call-ip6tables=0"},
	}
	for _, args := range sysctlCmds {
		cmd := exec.Command("sysctl", args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()
	}

	cmd := exec.Command("ip", "rule", "add", "fwmark", fwmark, "table", routeTable)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %w", err)
	}

	cmd = exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", routeTable)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %w", err)
	}

	return nil
}

func (f *Firewall) DeleteTproxyRoute(fwmark, routeTable string) error {
	cmd := exec.Command("ip", "rule", "del", "fwmark", fwmark, "table", routeTable)
	_ = cmd.Run()

	cmd = exec.Command("ip", "route", "flush", "table", routeTable)
	_ = cmd.Run()

	return nil
}

func detectFirewallBackend(cfg *config.Config) string {
	if isCommandAvailable("opkg") {
		switch cfg.ServerMode {
		case config.ServerModeTProxy:
			if isOpkgPackageInstalled("kmod-nft-tproxy") && isCommandAvailable("nft") {
				logrus.Info("Detected nftables backend (kmod-nft-tproxy installed)")
				return NFT
			}
			logrus.Info("Detected iptables backend (kmod-nft-tproxy not installed)")
			return IPT
		case config.ServerModeNFQueue:
			if isOpkgPackageInstalled("kmod-nft-queue") && isCommandAvailable("nft") {
				logrus.Info("Detected nftables backend (kmod-nft-queue installed)")
				return NFT
			}
			logrus.Info("Detected iptables backend (kmod-nft-queue not installed)")
			return IPT
		}
	}

	if isCommandAvailable("nft") {
		logrus.Info("Detected nftables backend (nft command available)")
		return NFT
	}

	if isCommandAvailable("iptables") {
		logrus.Info("Detected iptables backend (iptables command available)")
		return IPT
	}

	logrus.Warn("No firewall backend detected")
	return ""
}

func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func isOpkgPackageInstalled(pkg string) bool {
	cmd := exec.Command("opkg", "list-installed", pkg)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

func commandRunning(c string) bool {
	cmd := exec.Command("pgrep", "-f", c)
	err := cmd.Run()
	return err == nil
}

func shellclashExists() bool {
	if _, err := user.Lookup("shellclash"); err == nil {
		return true
	}
	if _, err := user.Lookup("shellcrash"); err == nil {
		return true
	}
	return false
}

func initSkipGids() {
	if commandRunning("openclash") {
		SKIP_GIDS += ",7890"
		SIDECAR = OC
	} else if commandRunning("ShellCrash") {
		SKIP_GIDS += ",65534"
		SIDECAR = SC
	} else if isOpkgPackageInstalled("luci-app-openclash") {
		SKIP_GIDS += ",7890"
		SIDECAR = OC
	} else if shellclashExists() {
		SKIP_GIDS += ",65534"
		SIDECAR = SC
	} else {
		SKIP_GIDS += ",7890"
		SIDECAR = OC
	}
}
