//go:build linux

package nfqueue

import (
	"strconv"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
)

const (
	table     = "mangle"
	chain     = "UA3F"
	jumpPoint = "POSTROUTING"
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

	// ensure netlink helper behind nfqueue server
	pos, exists := s.detectNfqueue(ipt)
	if !exists {
		err = ipt.Append(table, jumpPoint, JumpChain...)
	} else {
		err = ipt.Insert(table, jumpPoint, pos-1, JumpChain...)
	}
	if err != nil {
		return err
	}

	err = s.IptSetNfqueue(ipt)
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

func (s *Server) IptSetNfqueue(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, netfilter.IptRuleIgnoreReply...)
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
	var RuleIgnoreMark = []string{
		"-m", "connmark",
		"--mark", strconv.Itoa(int(s.NotHTTPCtMark)),
		"-j", "RETURN",
	}
	err = ipt.Append(table, chain, RuleIgnoreMark...)
	if err != nil {
		return err
	}
	var RuleNfqueue = []string{
		"-m", "conntrack",
		"--ctdir", "ORIGINAL",
		"--ctstate", "ESTABLISHED",
		"-m", "length",
		"--length", "41:0xffff",
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(int(s.nfqServer.QueueNum)),
		"--queue-bypass",
	}
	err = ipt.Append(table, chain, RuleNfqueue...)
	if err != nil {
		return err
	}
	return nil
}

// detect if iptables nfqueue rule exists and return nfqueue rule position
func (s *Server) detectNfqueue(ipt *iptables.IPTables) (pos int, exists bool) {
	rules, err := ipt.List(table, jumpPoint)
	if err != nil {
		return 0, false
	}
	for i, rule := range rules {
		if strings.Contains(rule, "NFQUEUE") {
			return i + 1, true
		}
	}
	return 0, false
}
