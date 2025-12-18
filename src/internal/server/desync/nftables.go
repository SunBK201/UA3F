//go:build linux

package desync

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

	s.NftSetDesync(tx, s.Nftable)

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

func (s *Server) NftSetDesync(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "DESYNC_QUEUE",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("mangle - 30")),
	}
	tx.Add(chain)

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnorePorts,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"ct state established",
			"ct direction original",
			"ip length > 41",
			fmt.Sprintf("ct bytes < %d", s.ReorderByte),
			fmt.Sprintf("ct packets < %d", s.ReorderPackets),
			fmt.Sprintf("counter queue num %d bypass", s.ReorderNfqServer.QueueNum),
		),
	})
}
