//go:build linux

package netlink

import (
	"context"
	"errors"
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

const (
	table = "mangle"
	chain = "POSTROUTING"
)

var RuleTTL = []string{
	"-j", "TTL",
	"--ttl-set", "64",
}

var RuleHookTCPSyn = []string{
	"-p", "tcp",
	"--tcp-flags", "SYN", "SYN",
	"-j", "NFQUEUE",
	"--queue-num", strconv.Itoa(netfilter.HELPER_QUEUE),
	"--queue-bypass",
}

var RuleIP = []string{
	"-p", "tcp",
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
		if netfilter.FlowOffloadEnabled() {
			err = s.IptSetTTLIngress(ipt)
			if err != nil {
				return err
			}
		}
	}
	if (s.cfg.DelTCPTimestamp || s.cfg.SetTCPInitialWindow) && !s.cfg.SetIPID {
		err = s.IptHookTCPSyn(ipt)
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
	_ = ipt.DeleteIfExists(table, chain, RuleHookTCPSyn...)
	if s.cfg.SetTTL {
		_ = s.NftCleanup()
	}
	return nil
}

func (s *Server) IptSetTTL(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleTTL...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) IptHookTCPSyn(ipt *iptables.IPTables) error {
	err := ipt.Append(table, chain, RuleHookTCPSyn...)
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

func (s *Server) IptSetTTLIngress(ipt *iptables.IPTables) error {
	if !netfilter.IsCommandAvailable("nft") {
		return errors.New("nft command not available")
	}

	nft, err := knftables.New(s.Nftable.Family, s.Nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.Nftable)
	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}

	lanDev, err := netfilter.GetLanDevice()
	if err != nil {
		return err
	}
	return s.NftSetTTLIngress(nft, s.Nftable, lanDev)
}
