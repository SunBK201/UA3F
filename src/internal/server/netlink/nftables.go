//go:build linux

package netlink

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
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

func (s *Server) NftDelTCPTS(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "DEL_TCPTS",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
		Hook:     knftables.PtrTo(knftables.PostroutingHook),
		Priority: knftables.PtrTo(knftables.ManglePriority),
	}
	tx.Add(chain)
	var rule *knftables.Rule
	if resetOptionAvailable() {
		rule = &knftables.Rule{
			Chain: chain.Name,
			Rule: knftables.Concat(
				"tcp option timestamp exists",
				"counter reset tcp option timestamp",
			),
		}
	} else {
		rule = &knftables.Rule{
			Chain: chain.Name,
			Rule: knftables.Concat(
				"tcp flags syn",
				fmt.Sprintf("counter queue num %d bypass", s.nfqServer.QueueNum),
			),
		}
	}
	tx.Add(rule)
}

func (s *Server) NftSetIP(tx *knftables.Transaction, table *knftables.Table) {
	chain := &knftables.Chain{
		Name:     "HELPER_QUEUE",
		Table:    table.Name,
		Type:     knftables.PtrTo(knftables.FilterType),
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

func resetOptionAvailable() bool {
	const TestName = "UA3F_TEST_RESET"
	table := &knftables.Table{
		Name:   TestName,
		Family: knftables.InetFamily,
	}
	nft, err := knftables.New(table.Family, table.Name)
	if err != nil {
		logrus.Errorf("resetOptionAvailable knftables.New: %v", err)
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
		logrus.Infof("tcp option reset is not available")
	}
	return err == nil
}
