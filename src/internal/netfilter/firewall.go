package netfilter

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os/exec"
	"os/user"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/gonetx/ipset"
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
	"192.168.0.0/16",
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
	NftRuleIgnoreNotBrLAN = knftables.Concat(
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
	initLanCidrs()
}

type Firewall struct {
	Nftable    *knftables.Table
	NftSetup   func() error
	NftCleanup func() error
	IptSetup   func() error
	IptCleanup func() error
}

func (f *Firewall) Setup(cfg *config.Config) (err error) {
	_ = f.Cleanup()
	backend := detectFirewallBackend(cfg)
	slog.Info("Setup firewall", slog.String("backend", backend))
	slog.Info("Exempt LAN CIDRs", slog.String("cidrs", fmt.Sprintf("%v", LAN_CIDRS)))
	switch backend {
	case NFT:
		if f.NftSetup == nil {
			return fmt.Errorf("nftables setup function is nil")
		}
		err = f.NftSetup()
	case IPT:
		if f.IptSetup == nil {
			return fmt.Errorf("iptables setup function is nil")
		}
		err = f.IptSetup()
	default:
		err = fmt.Errorf("unsupported or no firewall backend: %s", backend)
	}
	if err != nil {
		_ = f.Cleanup()
	}
	f.DumpNFTables()
	f.DumpIPTables()
	return err
}

func (f *Firewall) Cleanup() error {
	if f.NftCleanup != nil {
		_ = f.NftCleanup()
	}
	if f.IptCleanup != nil {
		_ = f.IptCleanup()
	}
	return nil
}

func (f *Firewall) DumpNFTables() {
	cmd := exec.Command("nft", "--handle", "list", "ruleset")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	slog.Info("nftables ruleset:\n" + string(output))
}

func (f *Firewall) DumpIPTables() {
	var tables = []string{"filter", "nat", "mangle", "raw"}
	for _, table := range tables {
		cmd := exec.Command("iptables", "-t", table, "-L", "-v", "-n", "--line-numbers")
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		slog.Debug(fmt.Sprintf("iptables table(%s):\n%s", table, string(output)))
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
	nftAvailable := IsCommandAvailable("nft")
	iptAvailable := IsCommandAvailable("iptables")
	nftTproxyAvailable := isOpkgPackageInstalled("kmod-nft-tproxy") && nftAvailable
	nftNfqueueAvailable := isOpkgPackageInstalled("kmod-nft-queue") && nftAvailable
	tproxyNeeded := cfg.ServerMode == config.ServerModeTProxy
	nfqueueNeeded := cfg.DelTCPTimestamp || cfg.SetIPID || cfg.ServerMode == config.ServerModeNFQueue

	selectNFT := func() bool {
		if !nftAvailable {
			return false
		}
		if nfqueueNeeded && !nftNfqueueAvailable {
			return false
		}
		if tproxyNeeded && !nftTproxyAvailable {
			return false
		}
		return true
	}

	selectIPT := func() bool {
		return iptAvailable
	}

	switch {
	case selectNFT():
		return NFT
	case selectIPT():
		return IPT
	default:
		slog.Warn("No firewall backend detected")
		return ""
	}
}

func getWanNexthops() ([]string, error) {
	out, err := exec.Command("ubus", "call", "network.interface.wan", "status").Output()
	if err != nil {
		return nil, err
	}
	var result struct {
		Route []struct {
			NextHop string `json:"nexthop"`
		} `json:"route"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		log.Fatal(err)
	}
	if len(result.Route) == 0 {
		return nil, errors.New("no route found for wan interface")
	}
	var nexthops []string
	for _, route := range result.Route {
		nexthops = append(nexthops, route.NextHop)
	}
	return nexthops, nil
}

func getLocalIPv4CIDRs() ([]string, error) {
	var cidrs []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ipv4 := ip.To4(); ipv4 != nil {
			cidrs = append(cidrs, fmt.Sprintf("%s/32", ipv4.String()))
		}
	}

	return cidrs, nil
}

func IsCommandAvailable(cmd string) bool {
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

func initLanCidrs() {
	// remove wan from lan cidrs
	nexthops, err := getWanNexthops()
	if err != nil {
		return
	}

	var lanRanges []net.IPNet
	for _, lan := range LAN_CIDRS {
		_, ipNet, err := net.ParseCIDR(lan)
		if err == nil {
			lanRanges = append(lanRanges, *ipNet)
		}
	}

	var wanIPs []net.IP
	for _, nh := range nexthops {
		if ip := net.ParseIP(nh); ip != nil {
			wanIPs = append(wanIPs, ip)
		}
	}

	remove := make(map[int]struct{})
	for i, lanNet := range lanRanges {
		for _, ip := range wanIPs {
			if lanNet.Contains(ip) {
				remove[i] = struct{}{}
				break
			}
		}
	}

	var updatedCIDRs []string
	for i, lanNet := range lanRanges {
		if _, ok := remove[i]; !ok {
			updatedCIDRs = append(updatedCIDRs, lanNet.String())
		}
	}

	LAN_CIDRS = updatedCIDRs

	localCIDRs, err := getLocalIPv4CIDRs()
	if err != nil {
		return
	}
	LAN_CIDRS = append(LAN_CIDRS, localCIDRs...)
}

func GetLanDevice() (string, error) {
	out, err := exec.Command("ubus", "call", "network.interface.lan", "status").Output()
	if err != nil {
		return "", err
	}
	var lanInterface struct {
		Device string `json:"device"`
	}
	if err := json.Unmarshal(out, &lanInterface); err != nil {
		return "", err
	}
	if lanInterface.Device == "" {
		return "", errors.New("no device found for lan interface")
	}
	// get real device if it's a bridge
	out, err = exec.Command("ubus", "call", "network.device", "status").Output()
	if err != nil {
		return "", err
	}
	var devices map[string]struct {
		Type    string   `json:"type"`
		Bridges []string `json:"bridge-members"`
	}
	if err := json.Unmarshal(out, &devices); err != nil {
		return "", err
	}
	dev, ok := devices[lanInterface.Device]
	if !ok {
		return "", fmt.Errorf("device %s not found", lanInterface.Device)
	}
	if dev.Type != "bridge" {
		return lanInterface.Device, nil
	}
	if len(dev.Bridges) == 0 {
		return "", fmt.Errorf("bridge %s has no members", lanInterface.Device)
	}
	return dev.Bridges[0], nil
}

func FlowOffloadEnabled() bool {
	cmd := exec.Command("nft", "list", "chain", string(knftables.InetFamily), "fw4", "forward")
	if output, err := cmd.CombinedOutput(); err == nil {
		if strings.Contains(string(output), "flow add") {
			return true
		}
	}

	ipt, err := iptables.New()
	if err != nil {
		return false
	}

	rules, err := ipt.List("filter", "forward")
	if err != nil {
		return false
	}
	for _, rule := range rules {
		if strings.Contains(rule, "FLOWOFFLOAD") {
			return true
		}
	}
	return false
}
