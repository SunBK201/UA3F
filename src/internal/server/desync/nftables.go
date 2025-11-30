//go:build linux

package desync

import (
	"context"
	"fmt"

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
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ip length > 41",
			"meta l4proto tcp",
			"ct state established",
			"ct direction original",
			fmt.Sprintf("ct bytes < %d", s.CtByte),
			fmt.Sprintf("ct packets < %d", s.CtPackets),
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}
