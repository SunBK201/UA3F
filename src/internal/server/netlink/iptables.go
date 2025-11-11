//go:build linux

package netlink

import (
	"github.com/coreos/go-iptables/iptables"
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
	"--queue-num", "10301",
	"--queue-bypass",
}

var RuleIP = []string{
	"-j", "NFQUEUE",
	"--queue-num", "10301",
	"--queue-bypass",
}

func (s *Server) iptSetup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	if s.cfg.SetTTL {
		err = IptSetTTL(ipt)
		if err != nil {
			return err
		}
	}
	if s.cfg.DelTCPTimestamp && !s.cfg.SetIPID {
		err = IptDelTCPTS(ipt)
		if err != nil {
			return err
		}
	}
	if s.cfg.SetIPID {
		err = IptSetIP(ipt)
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

func IptSetTTL(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleTTL...)
	if err != nil {
		return err
	}
	return nil
}

func IptDelTCPTS(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleDelTCPTS...)
	if err != nil {
		return err
	}
	return nil
}

func IptSetIP(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleIP...)
	if err != nil {
		return err
	}
	return nil
}
