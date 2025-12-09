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
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/daemon"
	"sigs.k8s.io/knftables"
)

const (
	NFT = "nft"
	IPT = "ipt"
)

const (
	LANSET       = "UA3F_LAN"
	SKIP_IPSET   = "UA3F_SKIP_IPSET"
	SKIP_PORTS   = "22,51080,51090"
	FAKEIP_RANGE = "198.18.0.0/16,198.18.0.1/15,28.0.0.1/8"
	HELPER_QUEUE = 10301
	DESYNC_QUEUE = 10901
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

var LAN6_CIDRS = []string{
	"::/128",
	"::1/128",
	"::ffff:0:0/96",
	"64:ff9b::/96",
	"2001:db8::/32",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
}

var SKIP_DOMAINS = []string{
	"st.dl.eccdnx.com",
	"st.dl.bscstorage.net",
	"st.dl.pinyuncloud.com",
	"dl.steam.clngaa.com",
	"cdn-ws.content.steamchina.com",
	"cdn-qc.content.steamchina.com",
	"cdn-ali.content.steamchina.com",
	"xz.pphimalayanrt.com",
	"lv.queniujq.cn",
	"alibaba.cdn.steampipe.steamcontent.com",
}

func init() {
	initSkipGids()
	initLanCidrs()
}

type Firewall struct {
	Nftable    *knftables.Table
	NftSetup   func() error
	NftCleanup func() error
	NftWatch   func()
	IptSetup   func() error
	IptCleanup func() error
	IptWatch   func()
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
		if f.NftWatch != nil {
			f.NftWatch()
		}
	case IPT:
		if f.IptSetup == nil {
			return fmt.Errorf("iptables setup function is nil")
		}
		err = f.IptSetup()
		if f.IptWatch != nil {
			f.IptWatch()
		}
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

	cmd = exec.Command("ip", "-6", "rule", "add", "fwmark", fwmark, "table", routeTable)
	if err := cmd.Run(); err != nil {
		slog.Warn("ip -6 rule add", slog.String("error", err.Error()))
	}

	cmd = exec.Command("ip", "-6", "route", "add", "local", "::/0", "dev", "lo", "table", routeTable)
	if err := cmd.Run(); err != nil {
		slog.Warn("ip -6 route add", slog.String("error", err.Error()))
	}

	return nil
}

func (f *Firewall) DeleteTproxyRoute(fwmark, routeTable string) error {
	cmd := exec.Command("ip", "rule", "del", "fwmark", fwmark, "table", routeTable)
	_ = cmd.Run()

	cmd = exec.Command("ip", "route", "flush", "table", routeTable)
	_ = cmd.Run()

	cmd = exec.Command("ip", "-6", "rule", "del", "fwmark", fwmark, "table", routeTable)
	_ = cmd.Run()

	cmd = exec.Command("ip", "-6", "route", "flush", "table", routeTable)
	_ = cmd.Run()

	return nil
}

func detectFirewallBackend(cfg *config.Config) string {
	isOpenwrt := daemon.IsOpenWrt()
	if isOpenwrt {
		slog.Info("Detected OpenWrt environment")
	}
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
		if !isOpenwrt {
			return true
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

func (f *Firewall) resolveDomains(domains []string) (v4 []string, v6 []string) {
	var ipv4Addrs []string
	var ipv6Addrs []string

	for _, domain := range domains {
		ips, err := net.LookupIP(domain)
		if err != nil {
			slog.Warn("net.LookupIP", slog.String("domain", domain), slog.Any("error", err))
			continue
		}
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				ipv4Addrs = append(ipv4Addrs, ipv4.String())
			} else if ipv6 := ip.To16(); ipv6 != nil {
				ipv6Addrs = append(ipv6Addrs, ipv6.String())
			}
		}
	}
	return ipv4Addrs, ipv6Addrs
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
