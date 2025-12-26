package match

import (
	"log/slog"
	"strings"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

type DomainSuffix struct {
	action       common.Action
	domainSuffix string
}

func (d *DomainSuffix) Type() common.RuleType {
	return common.RuleTypeDomainSuffix
}

func (d *DomainSuffix) Match(metadata *common.Metadata) bool {
	return strings.HasSuffix(metadata.Host(), d.domainSuffix)
}

func (d *DomainSuffix) Action() common.Action {
	return d.action
}

func NewDomainSuffix(rule *config.Rule) *DomainSuffix {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &DomainSuffix{
		action:       action,
		domainSuffix: rule.MatchValue,
	}
}
