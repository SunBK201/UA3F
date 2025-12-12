//go:build linux

package redirect

import (
	"context"
	"fmt"
	"time"

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
	s.NftSetLanIP6(tx, s.Nftable)
	s.NftSetSkipIP(tx, s.Nftable)
	s.NftSetSkipIP6(tx, s.Nftable)
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

func (s *Server) nftWatch() {
	go func() {
		_ = s.NftAddSkipDomains()

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = s.NftAddSkipDomains()
			case ip := <-s.SkipIpChan:
				if ip.To4() != nil {
					s.NftAddSkipIP(s.Nftable, []string{ip.String()})
				} else {
					s.NftAddSkipIP6(s.Nftable, []string{ip.String()})
				}
			}
		}
	}()
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
		Rule:  netfilter.NftRuleIgnoreNotBrLAN,
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
		Rule:  netfilter.NftRuleIgnoreLAN6,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreIP,
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule:  netfilter.NftRuleIgnoreIP6,
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
