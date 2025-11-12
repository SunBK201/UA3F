//go:build linux

package tproxy

import (
	"context"
	"fmt"
	"syscall"

	"github.com/sunbk201/ua3f/internal/netfilter"
	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	err := s.Firewall.AddTproxyRoute(s.tproxyFwMark, s.tproxyRouteTable)
	if err != nil {
		return err
	}

	nft, err := knftables.New(s.nftable.Family, s.nftable.Name)
	if err != nil {
		return err
	}

	tx := nft.NewTransaction()
	tx.Add(s.nftable)

	s.NftSetLanIP(tx, s.nftable)
	s.NftSetTproxy(tx, s.nftable)

	if err := nft.Run(context.TODO(), tx); err != nil {
		return err
	}
	return nil
}

func (s *Server) nftCleanup() error {
	_ = s.Firewall.DeleteTproxyRoute(s.tproxyFwMark, s.tproxyRouteTable)

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

func (s *Server) NftSetTproxy(tx *knftables.Transaction, table *knftables.Table) {
	if netfilter.SIDECAR == netfilter.SC {
		sidecar := &knftables.Chain{
			Name:     "UA3F_SIDECAR",
			Table:    table.Name,
			Type:     knftables.PtrTo(knftables.FilterType),
			Hook:     knftables.PtrTo(knftables.PreroutingHook),
			Priority: knftables.PtrTo(knftables.BaseChainPriority("mangle - 20")),
		}
		tx.Add(sidecar)
		tx.Add(&knftables.Rule{
			Chain: sidecar.Name,
			Rule: knftables.Concat(
				"meta l4proto tcp",
				"mark", s.tproxyFwMark,
				"mark set 7894",
				fmt.Sprintf("tproxy to 127.0.0.1:%d", s.cfg.Port),
				"counter accept",
			),
		})
	}

	prerouting := &knftables.Chain{
		Name:     "PREROUTING",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("filter + 20")),
	}
	tx.Add(prerouting)

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule:  netfilter.NftRuleIgnoreNotTCP,
	})

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule:  netfilter.NftRuleIgnoreReply,
	})

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule:  netfilter.NftRuleIgnoreFakeIP,
	})

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule:  netfilter.NftRuleIgnoreLAN,
	})

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule:  netfilter.NftRuleIgnorePorts,
	})

	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("mark %d", s.so_mark),
			"counter return",
		),
	})

	for _, mark := range s.ignoreMark {
		tx.Add(&knftables.Rule{
			Chain: prerouting.Name,
			Rule: knftables.Concat(
				fmt.Sprintf("mark {%s}", mark),
				"counter return",
			),
		})
	}

	// cap oc
	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"mark", s.tproxyFwMark,
			fmt.Sprintf("tproxy to 127.0.0.1:%d", s.cfg.Port),
			"counter accept",
		),
	})

	// default less hit. sc.
	tx.Add(&knftables.Rule{
		Chain: prerouting.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"mark set", s.tproxyFwMark,
			fmt.Sprintf("tproxy to 127.0.0.1:%d", s.cfg.Port),
			"counter accept",
		),
	})

	output := &knftables.Chain{
		Name:     "OUTPUT",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.RouteType),
		Hook:     knftables.PtrTo(knftables.OutputHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("filter + 20")),
	}
	tx.Add(output)

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule:  netfilter.NftRuleIgnoreNotTCP,
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule:  netfilter.NftRuleIgnoreReply,
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule:  netfilter.NftRuleIgnoreFakeIP,
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule:  netfilter.NftRuleIgnoreLAN,
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule:  netfilter.NftRuleIgnorePorts,
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("mark %d", s.so_mark),
			"counter return",
		),
	})

	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("meta skgid {%s}", netfilter.SKIP_GIDS),
			"counter return",
		),
	})

	// ghost oc
	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			fmt.Sprintf("meta skgid %d", syscall.Getgid()),
			"mark set", s.tproxyFwMark,
			"counter accept",
		),
	})

	// default tproxy mark. bypass sc pre pollution
	tx.Add(&knftables.Rule{
		Chain: output.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"mark set", s.tproxyFwMark,
			"counter accept",
		),
	})
}
