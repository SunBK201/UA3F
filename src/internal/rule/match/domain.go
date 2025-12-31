package match

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

type Domain struct {
	action common.Action
	domain string
}

func (d *Domain) Type() common.RuleType {
	return common.RuleTypeDomain
}

func (d *Domain) Match(metadata *common.Metadata) bool {
	return metadata.Host() == d.domain
}

func (d *Domain) Action() common.Action {
	return d.action
}

func NewDomain(rule *config.Rule) *Domain {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &Domain{
		action: action,
		domain: rule.MatchValue,
	}
}
