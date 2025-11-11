//go:build linux

package netlink

import (
	"context"
	"fmt"

	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	nft, err := knftables.New(s.nftable.Family, s.nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.nftable)

	if s.cfg.SetTTL {
		s.NftSetTTL(tx, s.nftable)
	}
	if s.cfg.DelTCPTimestamp && !s.cfg.SetIPID {
		s.NftDelTCPTS(tx, s.nftable)
	}
	if s.cfg.SetIPID {
		s.NftSetIP(tx, s.nftable)
	}

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (s *Server) nftCleanup() error {
	nft, err := knftables.New(s.nftable.Family, s.nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Delete(s.nftable)

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (s *Server) NftSetTTL(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "TTL64",
		Type:     knftables.PtrTo(knftables.FilterType),
		Table:    table.Name,
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

func (s *Server) NftDelTCPTS(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "HELPER_QUEUE",
		Type:     knftables.PtrTo(knftables.FilterType),
		Table:    table.Name,
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"tcp flags syn",
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}

func (s *Server) NftSetIP(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "HELPER_QUEUE",
		Type:     knftables.PtrTo(knftables.FilterType),
		Table:    table.Name,
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	rule := &knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}
