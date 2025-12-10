//go:build linux

package netlink

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	if !s.cfg.SetTTL && !s.cfg.DelTCPTimestamp && !s.cfg.SetIPID {
		slog.Info("No packet modification features enabled, skipping nftables setup")
		return nil
	}

	nft, err := knftables.New(s.Nftable.Family, s.Nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.Nftable)

	if s.cfg.SetTTL {
		s.NftSetTTL(tx, s.Nftable)
	}
	if (s.cfg.DelTCPTimestamp || s.cfg.SetTCPInitialWindow) && !s.cfg.SetIPID {
		s.NftHookTCPSyn(tx, s.Nftable)
	}
	if s.cfg.SetIPID {
		s.NftHookIP(tx, s.Nftable)
	}

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}

	if s.cfg.SetTTL && netfilter.FlowOffloadEnabled() {
		lanDev, err := netfilter.GetLanDevice()
		if err != nil {
			slog.Warn("nftSetup netfilter.GetLanDevice", slog.Any("error", err))
		} else {
			err = s.NftSetTTLIngress(nft, s.Nftable, lanDev)
			if err != nil {
				slog.Warn("NftSetTTLIngress", slog.Any("error", err))
			}
		}
	}
	return nil
}

func (s *Server) nftCleanup() error {
	nft, err := knftables.New(s.Nftable.Family, s.Nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Delete(s.Nftable)

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (s *Server) NftSetTTL(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "TTL64",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ip ttl set 64",
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}

func (s *Server) NftSetTTLIngress(nft knftables.Interface, table *knftables.Table, device string) error {
	tx := nft.NewTransaction()

	chain := &knftables.Chain{
		Name:     "TTL64_INGRESS",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.IngressHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
		Device:   knftables.PtrTo(device),
	}
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ip ttl set 65",
		),
	}
	tx.Add(chain)
	tx.Add(rule)

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (s *Server) NftHookTCPSyn(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "HOOK_TCP_SYN",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	tx.Add(chain)

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnorePorts,
	})
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"tcp flags syn",
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	})
}

func (s *Server) NftHookIP(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "HELPER_QUEUE",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	tx.Add(chain)

	if s.cfg.SetTCPInitialWindow || s.cfg.DelTCPTimestamp {
		tx.Add(&knftables.Rule{
			Chain: chain.Name,
			Rule: knftables.Concat(
				"tcp flags syn",
				fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
			),
		})
	}
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ip id != 0",
			"meta l4proto tcp",
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	})
}

// unused currently
func ResetOptionAvailable() bool {
	const TestName = "UA3F_TEST_RESET"
	table := &knftables.Table{
		Name:   TestName,
		Family: knftables.InetFamily,
	}
	nft, err := knftables.New(table.Family, table.Name)
	if err != nil {
		slog.Error("ResetOptionAvailable knftables.New", slog.Any("error", err))
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
			"tcp option timestamp exists",
			"counter reset tcp option timestamp",
		),
	}
	tx.Add(table)
	tx.Add(chain)
	tx.Add(rule)
	err = nft.Check(context.TODO(), tx)
	if err != nil {
		slog.Info("tcp option reset is not available")
	}
	return err == nil
}
