//go:build linux

package netlink

import (
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
)

const (
	table = "mangle"
	chain = "POSTROUTING"
)

var RuleTTL = []string{
	"-j", "TTL",
	"--ttl-set", "64",
}

var RuleDelTCPTS = []string{
	"-p", "tcp",
	"--tcp-flags", "SYN", "SYN",
	"-j", "NFQUEUE",
	"--queue-num", strconv.Itoa(netfilter.HELPER_QUEUE),
	"--queue-bypass",
}

var RuleIP = []string{
	"-j", "NFQUEUE",
	"--queue-num", strconv.Itoa(netfilter.HELPER_QUEUE),
	"--queue-bypass",
}

var RuleRstTimestamp = []string{
	"-p", "tcp",
	"--tcp-option", "8",
	"-j", "TCPOPTSTRIP",
	"--strip-options", "timestamp",
}

func (s *Server) iptSetup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	if s.cfg.SetTTL {
		err = s.IptSetTTL(ipt)
		if err != nil {
			return err
		}
	}
	if s.cfg.DelTCPTimestamp && !s.cfg.SetIPID {
		err = s.IptDelTCPTS(ipt)
		if err != nil {
			return err
		}
	}
	if s.cfg.SetIPID {
		err = s.IptSetIP(ipt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) iptCleanup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	_ = ipt.DeleteIfExists(table, chain, RuleTTL...)
	_ = ipt.DeleteIfExists(table, chain, RuleIP...)
	_ = ipt.DeleteIfExists(table, chain, RuleDelTCPTS...)
	return nil
}

func (s *Server) IptSetTTL(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleTTL...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) IptDelTCPTS(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleRstTimestamp...)
	if err == nil {
		return nil
	}

	err = ipt.Append(table, chain, RuleDelTCPTS...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) IptSetIP(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleIP...)
	if err != nil {
		return err
	}
	return nil
}
