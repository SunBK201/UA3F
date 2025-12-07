package netfilter

import (
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/gonetx/ipset"
)

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
	IptRuleIgnoreIP = []string{
		"-m", "set",
		"--match-set", SKIP_IPSET, "dst",
		"-j", "RETURN",
	}
	IptRuleIgnorePorts = []string{
		"-p", "tcp",
		"-m", "multiport",
		"--dports", SKIP_PORTS,
		"-j", "RETURN",
	}
)

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

func (f *Firewall) IptSetSkipIP() error {
	if err := ipset.Check(); err != nil {
		return err
	}
	_, err := ipset.New(
		SKIP_IPSET,
		ipset.HashIp,
		ipset.Family(ipset.Inet),
		ipset.Timeout(time.Hour),
		ipset.Exist(false),
	)
	if err != nil {
		return err
	}

	_ = f.IptAddSkipDomains()

	return nil
}

func (f *Firewall) IptDeleteSkipIP() error {
	return ipset.Destroy(SKIP_IPSET)
}

func (f *Firewall) IptAddSkipIP(ip string) error {
	if err := ipset.Check(); err != nil {
		return err
	}
	set, err := ipset.New(
		SKIP_IPSET,
		ipset.HashIp,
		ipset.Family(ipset.Inet),
		ipset.Timeout(time.Hour),
		ipset.Exist(true),
	)
	if err != nil {
		return err
	}

	if err := set.Add(ip); err != nil {
		return err
	}
	return nil
}

func (f *Firewall) IptAddSkipDomains() error {
	if err := ipset.Check(); err != nil {
		return err
	}
	set, err := ipset.New(
		SKIP_IPSET,
		ipset.HashIp,
		ipset.Family(ipset.Inet),
		ipset.Timeout(time.Hour),
		ipset.Exist(true),
	)
	if err != nil {
		return err
	}

	v4Addrs, v6Addrs := f.resolveDomains(SKIP_DOMAINS)
	for _, addr := range v4Addrs {
		if err := set.Add(addr); err != nil {
			return err
		}
	}
	for _, addr := range v6Addrs {
		if err := set.Add(addr); err != nil {
			return err
		}
	}
	return nil
}
