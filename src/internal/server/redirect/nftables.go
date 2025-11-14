//go:build linux

package redirect

import (
	"context"
	"fmt"

	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	nft, err := knftables.New(s.Nftable.Family, s.Nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.Nftable)

	s.NftSetLanIP(tx, s.Nftable)
	s.NftSetRedirect(tx, s.Nftable)

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
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

func (s *Server) NftSetRedirect(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "PREROUTING",
		Type:     knftables.PtrTo(knftables.NATType),
		Table:    table.Name,
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("dstnat - 20")),
	}
	tx.Add(chain)

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreBrLAN,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreNotTCP,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreReply,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreLAN,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnorePorts,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("mark %d", s.so_mark),
			"counter return",
		),
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"tcp dport != {22}",
			fmt.Sprintf("counter redirect to :%d", s.Cfg.Port),
		),
	})
}
