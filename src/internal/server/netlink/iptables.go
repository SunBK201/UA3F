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

var RuleIP = []string{
	"-j", "NFQUEUE",
	"--queue-num", "10301",
	"--queue-bypass",
}

var RuleDelTCPTS = []string{
	"-p", "tcp",
	"--tcp-flags", "SYN,RST,ACK,FIN",
	"-j", "NFQUEUE",
	"--queue-num", "10301",
	"--queue-bypass",
}

func (s *Server) iptSetup() error {
	_ = s.iptCleanup()

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
	err = ipt.DeleteIfExists(table, chain, RuleDelTCPTS...)
	if err != nil {
		return err
	}
	return nil
}

// IptSetTTL creates a chain that sets TTL to 64 for IPv4 packets
func IptSetTTL(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleTTL...)
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

func IptDelTCPTS(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleDelTCPTS...)
	if err != nil {
		return err
	}
	return nil
}
