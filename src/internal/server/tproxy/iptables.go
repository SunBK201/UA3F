//go:build linux

package tproxy

import (
	"strconv"
	"strings"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sunbk201/ua3f/internal/netfilter"
)

const (
	table            = "mangle"
	chainPre         = "UA3F"
	chainOut         = "UA3F_OUTPUT"
	chainSidecar     = "UA3F_SIDECAR"
	jumpPointPre     = "PREROUTING"
	jumpPointOut     = "OUTPUT"
	jumpPointSidecar = "PREROUTING"
)

var FakeIPs = []string{
	"198.18.0.0/16",
	"28.0.0.1/8",
	"198.18.0.1/15",
}

var JumpChainPre = []string{
	"-p", "tcp",
	"-j", chainPre,
}

var JumpChainOut = []string{
	"-p", "tcp",
	"-j", chainOut,
}

var JumpChainSidecar = []string{
	"-p", "tcp",
	"-j", chainSidecar,
}

func (s *Server) iptSetup() error {
	err := s.Firewall.AddTproxyRoute(s.tproxyFwMark, s.tproxyRouteTable)
	if err != nil {
		return err
	}

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}

	err = s.IptSetLanIP()
	if err != nil {
		return err
	}

	if netfilter.SIDECAR == netfilter.SC {
		err = ipt.NewChain(table, chainSidecar)
		if err != nil {
			return err
		}
		err = ipt.Insert(table, jumpPointSidecar, 1, JumpChainSidecar...)
		if err != nil {
			return err
		}
	}

	err = ipt.NewChain(table, chainPre)
	if err != nil {
		return err
	}

	err = ipt.NewChain(table, chainOut)
	if err != nil {
		return err
	}

	err = ipt.Append(table, jumpPointPre, JumpChainPre...)
	if err != nil {
		return err
	}

	err = ipt.Append(table, jumpPointOut, JumpChainOut...)
	if err != nil {
		return err
	}

	err = s.IptSetTproxy(ipt)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) iptCleanup() error {
	_ = s.Firewall.DeleteTproxyRoute(s.tproxyFwMark, s.tproxyRouteTable)
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	ipt.Delete(table, jumpPointPre, JumpChainPre...)
	ipt.Delete(table, jumpPointOut, JumpChainOut...)
	ipt.Delete(table, jumpPointSidecar, JumpChainSidecar...)
	ipt.ClearAndDeleteChain(table, chainPre)
	ipt.ClearAndDeleteChain(table, chainOut)
	ipt.ClearAndDeleteChain(table, chainSidecar)
	s.IptDeleteLanIP()
	return nil
}

func (s *Server) IptSetTproxy(ipt *iptables.IPTables) error {
	if netfilter.SIDECAR == netfilter.SC {
		var RuleSidecar = []string{
			"-p", "tcp",
			"-m", "mark",
			"--mark", s.tproxyFwMark,
			"-j", "TPROXY",
			"--on-ip", "127.0.0.1",
			"--on-port", strconv.Itoa(s.cfg.Port),
			"--tproxy-mark", "7894",
		}
		err := ipt.Append(table, chainSidecar, RuleSidecar...)
		if err != nil {
			return err
		}
	}

	// PREROUTING
	err := ipt.Append(table, chainPre, netfilter.IptRuleIgnoreReply...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chainPre, netfilter.IptRuleIgnoreLAN...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chainPre, netfilter.IptRuleIgnorePorts...)
	if err != nil {
		return err
	}

	var RuleIgnoreSOMark = []string{
		"-m", "mark",
		"--mark", strconv.Itoa(s.so_mark),
		"-j", "RETURN",
	}
	err = ipt.Append(table, chainPre, RuleIgnoreSOMark...)
	if err != nil {
		return err
	}

	for _, imark := range s.ignoreMark {
		var RuleIgnoreMark = []string{
			"-m", "mark",
			"--mark", imark,
			"-j", "RETURN",
		}
		err = ipt.Append(table, chainPre, RuleIgnoreMark...)
		if err != nil {
			return err
		}
	}

	for _, ipr := range FakeIPs {
		var RuleIgnoreFakeIP = []string{
			"-d", ipr,
			"-j", "RETURN",
		}
		err = ipt.Append(table, chainPre, RuleIgnoreFakeIP...)
		if err != nil {
			return err
		}
	}

	var RuleTproxyOC = []string{
		"-p", "tcp",
		"-m", "mark",
		"--mark", s.tproxyFwMark,
		"-j", "TPROXY",
		"--on-ip", "127.0.0.1",
		"--on-port", strconv.Itoa(s.cfg.Port),
	}
	err = ipt.Append(table, chainPre, RuleTproxyOC...)
	if err != nil {
		return err
	}

	var RuleTproxy = []string{
		"-p", "tcp",
		"-j", "TPROXY",
		"--on-ip", "127.0.0.1",
		"--on-port", strconv.Itoa(s.cfg.Port),
		"--tproxy-mark", s.tproxyFwMark,
	}
	err = ipt.Append(table, chainPre, RuleTproxy...)
	if err != nil {
		return err
	}

	// OUTPUT
	err = ipt.Append(table, chainOut, netfilter.IptRuleIgnoreReply...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chainOut, netfilter.IptRuleIgnoreLAN...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chainOut, netfilter.IptRuleIgnorePorts...)
	if err != nil {
		return err
	}
	err = ipt.Append(table, chainOut, RuleIgnoreSOMark...)
	if err != nil {
		return err
	}

	for _, ipr := range FakeIPs {
		var RuleIgnoreFakeIP = []string{
			"-d", ipr,
			"-j", "RETURN",
		}
		err = ipt.Append(table, chainOut, RuleIgnoreFakeIP...)
		if err != nil {
			return err
		}
	}

	skipGids := strings.Split(netfilter.SKIP_GIDS, ",")
	for _, gid := range skipGids {
		gid = strings.TrimSpace(gid)
		if gid == "" {
			continue
		}
		var RuleIgnoreGID = []string{
			"-m", "owner",
			"--gid-owner", gid,
			"-j", "RETURN",
		}
		err = ipt.Append(table, chainOut, RuleIgnoreGID...)
		if err != nil {
			return err
		}
	}

	var RuleTproxyOutputOC = []string{
		"-p", "tcp",
		"-m", "owner",
		"--gid-owner", strconv.Itoa(syscall.Getgid()),
		"-j", "MARK",
		"--set-mark", s.tproxyFwMark,
	}
	err = ipt.Append(table, chainOut, RuleTproxyOutputOC...)
	if err != nil {
		return err
	}

	var RuleTproxyOutput = []string{
		"-p", "tcp",
		"-j", "MARK",
		"--set-mark", s.tproxyFwMark,
	}
	err = ipt.Append(table, chainOut, RuleTproxyOutput...)
	if err != nil {
		return err
	}

	return nil
}
