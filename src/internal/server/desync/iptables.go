//go:build linux

package desync

import (
	"strconv"

	"github.com/coreos/go-iptables/iptables"
)

const (
	table           = "mangle"
	reorderChain    = "UA3F_REORDER_DESYNC"
	injectChain     = "UA3F_INJECT_DESYNC"
	jumpPoint       = "POSTROUTING"
	injectJumpPoint = "PREROUTING"
)

var (
	JumpReorderChain = []string{
		"-p", "tcp",
		"-j", reorderChain,
	}
	JumpInjectChain = []string{
		"-p", "tcp",
		"-j", injectChain,
	}
)

func (s *Server) iptSetup() error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}

	if s.cfg.Desync.Reorder {
		err = ipt.NewChain(table, reorderChain)
		if err != nil {
			return err
		}
		err = ipt.Append(table, jumpPoint, JumpReorderChain...)
		if err != nil {
			return err
		}
		err = s.IptSetDesyncReorder(ipt)
		if err != nil {
			return err
		}
	}
	if s.cfg.Desync.Inject {
		err = ipt.NewChain(table, injectChain)
		if err != nil {
			return err
		}
		err = ipt.Append(table, injectJumpPoint, JumpInjectChain...)
		if err != nil {
			return err
		}
		err = s.IptSetDesyncInject(ipt)
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
	ipt.Delete(table, jumpPoint, JumpReorderChain...)
	ipt.Delete(table, injectJumpPoint, JumpInjectChain...)
	ipt.ClearAndDeleteChain(table, injectChain)
	ipt.ClearAndDeleteChain(table, reorderChain)
	return nil
}

func (s *Server) IptSetDesyncInject(ipt *iptables.IPTables) error {
	var RuleDesync = []string{
		"-p", "tcp",
		"--tcp-flags", "SYN,ACK", "SYN,ACK",
		"-m", "conntrack",
		"--ctdir", "REPLY",
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(int(s.InjectNfqServer.QueueNum)),
		"--queue-bypass",
	}
	err := ipt.Append(table, injectChain, RuleDesync...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) IptSetDesyncReorder(ipt *iptables.IPTables) error {
	var RuleIgnoreSOMark = []string{
		"-m", "mark",
		"--mark", strconv.Itoa(s.InjectMark),
		"-j", "RETURN",
	}
	err := ipt.Append(table, reorderChain, RuleIgnoreSOMark...)
	if err != nil {
		return err
	}
	var RuleDesync = []string{
		"-p", "tcp",
		"-m", "conntrack",
		"--ctdir", "ORIGINAL",
		"--ctstate", "ESTABLISHED",
		"-m", "connbytes",
		"--connbytes-dir", "original",
		"--connbytes-mode", "bytes",
		"--connbytes", "0:" + strconv.Itoa(int(s.ReorderByte)),
		"-m", "connbytes",
		"--connbytes-dir", "original",
		"--connbytes-mode", "packets",
		"--connbytes", "0:" + strconv.Itoa(int(s.ReorderPackets)),
		"-m", "length",
		"--length", "41:0xffff",
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(int(s.ReorderNfqServer.QueueNum)),
		"--queue-bypass",
	}
	err = ipt.Append(table, reorderChain, RuleDesync...)
	if err != nil {
		return err
	}
	return nil
}
