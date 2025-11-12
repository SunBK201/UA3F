//go:build linux

package redirect

import (
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
)

const (
	table     = "nat"
	chain     = "UA3F"
	jumpPoint = "PREROUTING"
)

var JumpChain = []string{
	"-p", "tcp",
	"-j", chain,
}

func (s *Server) iptSetup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}

	err = s.IptSetLanIP()
	if err != nil {
		return err
	}

	err = ipt.NewChain(table, chain)
	if err != nil {
		return err
	}

	err = ipt.Insert(table, jumpPoint, 1, JumpChain...)
	if err != nil {
		return err
	}

	err = s.IptSetRedirect(ipt)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) iptCleanup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	ipt.Delete(table, jumpPoint, JumpChain...)
	ipt.ClearAndDeleteChain(table, chain)
	s.IptDeleteLanIP()
	return nil
}

func (s *Server) IptSetRedirect(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, netfilter.IptRuleIgnoreBrLAN...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chain, netfilter.IptRuleIgnoreReply...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chain, netfilter.IptRuleIgnoreLAN...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chain, netfilter.IptRuleIgnorePorts...)
	if err != nil {
		return err
	}
	var RuleIgnoreSOMark = []string{
		"-m", "mark",
		"--mark", strconv.Itoa(s.so_mark),
		"-j", "RETURN",
	}
	err = ipt.Append(table, chain, RuleIgnoreSOMark...)
	if err != nil {
		return err
	}
	var RuleRedirect = []string{
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", strconv.Itoa(s.cfg.Port),
	}
	err = ipt.Append(table, chain, RuleRedirect...)
	if err != nil {
		return err
	}
	return nil
}
