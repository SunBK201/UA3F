package netfilter

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"sigs.k8s.io/knftables"
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
	NftRuleIgnoreLAN6 = knftables.Concat(
		fmt.Sprintf("ip6 daddr @%s", LANSET+"_6"),
		"return",
	)
	NftRuleIgnoreIP = knftables.Concat(
		fmt.Sprintf("ip daddr @%s", SKIP_IPSET),
		"return",
	)
	NftRuleIgnoreIP6 = knftables.Concat(
		fmt.Sprintf("ip6 daddr @%s", SKIP_IPSET+"_6"),
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

func (f *Firewall) DumpNFTables() {
	cmd := exec.Command("nft", "--handle", "list", "ruleset")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	slog.Info("nftables ruleset:\n" + string(output))
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

func (f *Firewall) NftSetLanIP6(tx *knftables.Transaction, table *knftables.Table) {
	ipset := &knftables.Set{
		Name:   LANSET + "_6",
		Table:  table.Name,
		Family: table.Family,
		Type:   "ipv6_addr",
		Flags: []knftables.SetFlag{
			knftables.IntervalFlag,
		},
		AutoMerge: knftables.PtrTo(true),
	}
	tx.Add(ipset)

	for _, cidr := range LAN6_CIDRS {
		ip6lan := &knftables.Element{
			Table:  table.Name,
			Family: table.Family,
			Set:    ipset.Name,
			Key:    []string{cidr},
		}
		tx.Add(ip6lan)
	}
}

func (f *Firewall) NftSetSkipIP(tx *knftables.Transaction, table *knftables.Table) {
	ipset := &knftables.Set{
		Name:   SKIP_IPSET,
		Table:  table.Name,
		Family: table.Family,
		Type:   "ipv4_addr",
		Flags: []knftables.SetFlag{
			knftables.TimeoutFlag,
		},
		Timeout: knftables.PtrTo(3600 * time.Second),
	}
	tx.Add(ipset)
}

func (f *Firewall) NftSetSkipIP6(tx *knftables.Transaction, table *knftables.Table) {
	ipset := &knftables.Set{
		Name:   SKIP_IPSET + "_6",
		Table:  table.Name,
		Family: table.Family,
		Type:   "ipv6_addr",
		Flags: []knftables.SetFlag{
			knftables.TimeoutFlag,
		},
		Timeout: knftables.PtrTo(3600 * time.Second),
	}
	tx.Add(ipset)
}

func (f *Firewall) NftAddSkipIP(table *knftables.Table, addrs []string) error {
	nft, err := knftables.New(table.Family, table.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	for _, addr := range addrs {
		element := &knftables.Element{
			Table:  table.Name,
			Family: table.Family,
			Set:    SKIP_IPSET,
			Key:    []string{addr},
		}
		tx.Add(element)
	}

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (f *Firewall) NftAddSkipIP6(table *knftables.Table, addrs []string) error {
	nft, err := knftables.New(table.Family, table.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	for _, addr := range addrs {
		element := &knftables.Element{
			Table:  table.Name,
			Family: table.Family,
			Set:    SKIP_IPSET + "_6",
			Key:    []string{addr},
		}
		tx.Add(element)
	}

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (f *Firewall) NftAddSkipDomains() error {
	v4Addrs, v6Addrs := f.resolveDomains(SKIP_DOMAINS)

	if len(v4Addrs) > 0 {
		if err := f.NftAddSkipIP(f.Nftable, v4Addrs); err != nil {
			slog.Warn("f.NftAddSkipIP", slog.Any("error", err))
			return err
		}
	}
	if len(v6Addrs) > 0 {
		if err := f.NftAddSkipIP6(f.Nftable, v6Addrs); err != nil {
			slog.Warn("f.NftAddSkipIP6", slog.Any("error", err))
			return err
		}
	}
	return nil
}

func NftIHAvailable() bool {
	const TestName = "UA3F_TEST_IH"
	table := &knftables.Table{
		Name:   TestName,
		Family: knftables.InetFamily,
	}
	nft, err := knftables.New(table.Family, table.Name)
	if err != nil {
		slog.Error("NftIHAvailable knftables.New", slog.Any("error", err))
		return false
	}
	tx := nft.NewTransaction()
	chain := &knftables.Chain{
		Name:     TestName,
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"ct direction original",
			"ct state established",
			"@ih,0,8 & 0 == 0",
			"counter accept",
		),
	}
	tx.Add(table)
	tx.Add(chain)
	tx.Add(rule)
	err = nft.Check(context.TODO(), tx)
	if err != nil {
		slog.Info("@ih match not available", slog.Any("error", err))
	}
	return err == nil
}
