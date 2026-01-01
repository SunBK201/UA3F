//go:build linux

package desync

import (
	"context"
	"fmt"
	"strings"

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

	if s.cfg.Desync.Reorder {
		s.NftSetDesyncReorder(tx, s.Nftable)
	}
	if s.cfg.Desync.Inject {
		s.NftSetLanIP(tx, s.Nftable)
		s.NftSetLanIP6(tx, s.Nftable)
		s.NftSetDesyncInject(tx, s.Nftable)
	}

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

func (s *Server) NftSetDesyncInject(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "DESYNC_INJECT_QUEUE",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.BaseChainPriority("mangle - 30")),
	}
	tx.Add(chain)

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("ip saddr @%s", netfilter.LANSET),
			"return",
		),
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("ip6 saddr @%s", netfilter.LANSET+"_6"),
			"return",
		),
	})

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			fmt.Sprintf("tcp sport { %s }", netfilter.SKIP_PORTS),
			"return",
		),
	})

	if len(s.DesyncPorts) > 0 {
		ports := make([]string, 0, len(s.DesyncPorts))
		for _, p := range s.DesyncPorts {
			ports = append(ports, fmt.Sprintf("%d", p))
		}
		tx.Add(&knftables.Rule{
			Chain: chain.Name,
			Rule: knftables.Concat(
				"meta l4proto tcp",
				fmt.Sprintf("tcp sport != { %s }", strings.Join(ports, ",")),
				"counter return",
			),
		})
	}

	tx.Add(&knftables.Rule{
		Chain: chain.Name,
		Rule: knftables.Concat(
			"meta l4proto tcp",
			"ct direction reply",
			"tcp flags syn,ack / syn,ack",
			fmt.Sprintf("counter queue num %d bypass", s.InjectNfqServer.QueueNum),
		),
	})
}

func (s *Server) NftSetDesyncReorder(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "DESYNC_REORDER_QUEUE",
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
			fmt.Sprintf("mark %d", s.InjectMark),
			"counter return",
		),
	})

	if len(s.DesyncPorts) > 0 {
		ports := make([]string, 0, len(s.DesyncPorts))
		for _, p := range s.DesyncPorts {
			ports = append(ports, fmt.Sprintf("%d", p))
		}
		tx.Add(&knftables.Rule{
			Chain: chain.Name,
			Rule: knftables.Concat(
				"meta l4proto tcp",
				fmt.Sprintf("tcp dport != { %s }", strings.Join(ports, ",")),
				"counter return",
			),
		})
	}

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
