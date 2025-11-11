package netlink

import (
	"context"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/knftables"
)

func (s *Server) nftSetup() error {
	// Clean up existing table if any
	_ = s.nftCleanup()

	nft, err := knftables.New(s.nftable.Family, s.nftable.Name)
	if err != nil {
		logrus.Errorf("Failed to create nftables table: %v", err)
		return err
	}
	tx := nft.NewTransaction()
	tx.Add(s.nftable)

	if s.cfg.SetTTL {
		NftSetTTL(tx, s.nftable)
	}
	if s.cfg.DelTCPTimestamp && !s.cfg.SetIPID {
		NftDelTCPTS(tx, s.nftable)
	}
	if s.cfg.SetIPID {
		NftSetIP(tx, s.nftable)
	}

	if err := nft.Run(context.TODO(), tx); err != nil {
		logrus.Errorf("Failed to run nftables transaction: %v", err)
		return err
	}

	logrus.Info("Nftables setup completed")
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

// NftSetTTL creates a chain that sets TTL to 64 for IPv4 packets
func NftSetTTL(tx *knftables.Transaction, table *knftables.Table) {
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

func NftSetIP(tx *knftables.Transaction, table *knftables.Table) {
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
			"counter queue num 10301 bypass",
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}

func NftDelTCPTS(tx *knftables.Transaction, table *knftables.Table) {
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
			"tcp flags syn counter queue num 10301 bypass",
		),
	}
	tx.Add(chain)
	tx.Add(rule)
}
