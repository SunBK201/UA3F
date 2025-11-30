//go:build linux

package desync

import (
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
)

const (
	table     = "mangle"
	chain     = "UA3F_DESYNC"
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

	err = ipt.NewChain(table, chain)
	if err != nil {
		return err
	}

	err = ipt.Append(table, jumpPoint, JumpChain...)
	if err != nil {
		return err
	}

	return s.IptSetRuleDesync(ipt)
}

func (s *Server) iptCleanup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	ipt.Delete(table, jumpPoint, JumpChain...)
	ipt.ClearAndDeleteChain(table, chain)
	return nil
}

func (s *Server) IptSetRuleDesync(ipt *iptables.IPTables) error {
	var RuleDesync = []string{
		"-p", "tcp",
		"-m", "conntrack",
		"--ctdir", "ORIGINAL",
		"--ctstate", "ESTABLISHED",
		"-m", "connbytes",
		"--connbytes-dir", "original",
		"--connbytes-mode", "bytes",
		"--connbytes", "0:" + strconv.Itoa(int(s.CtByte)),
		"-m", "connbytes",
		"--connbytes-dir", "original",
		"--connbytes-mode", "packets",
		"--connbytes", "0:" + strconv.Itoa(int(s.CtPackets)),
		"-m", "length",
		"--length", "41:0xffff",
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(netfilter.DESYNC_QUEUE),
		"--queue-bypass",
	}
	err := ipt.Append(table, chain, RuleDesync...)
	if err != nil {
		return err
	}
	return nil
}
