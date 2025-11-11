//go:build linux

package nfqueue

import (
	"context"
	"fmt"

	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	nft, err := knftables.New(s.nftable.Family, s.nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.nftable)

	s.NftSetLanIP(tx, s.nftable)
	s.NftSetNfqueue(tx, s.nftable)

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

func (s *Server) NftSetNfqueue(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "POSTROUTING",
		Type:     knftables.PtrTo(knftables.FilterType),
		Table:    table.Name,
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("mangle - 20")),
	}
	tx.Add(chain)

	// nft add rule ip $NFT_TABLE postrouting meta l4proto != tcp counter return
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"meta l4proto != tcp",
			"return",
		),
	})

	// nft add rule ip $NFT_TABLE postrouting ct direction reply counter return
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ct direction reply",
			"return",
		),
	})

	// nft add rule ip $NFT_TABLE postrouting ip daddr @$UA3F_LANSET counter return
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("ip daddr @%s", netfilter.LANSET),
			"return",
		),
	})

	// nft add rule ip $NFT_TABLE postrouting tcp dport {$SKIP_PORTS} return
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("tcp dport {%s}", netfilter.SKIP_PORTS),
			"return",
		),
	})

	// nft add rule ip $NFT_TABLE postrouting ct mark 201 counter return
	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("ct mark %d", s.NotHTTPMark),
			"counter return",
		),
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"ct direction original",
			"ct state established",
			"ip length > 40",
			fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
		),
	})
}
