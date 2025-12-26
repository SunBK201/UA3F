package match

import (
	"log/slog"
	"strings"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

type DomainKeyword struct {
	action        common.Action
	domainKeyword string
}

func (d *DomainKeyword) Type() common.RuleType {
	return common.RuleTypeDomainKeyword
}

func (d *DomainKeyword) Match(metadata *common.Metadata) bool {
	return strings.Contains(metadata.Host(), d.domainKeyword)
}

func (d *DomainKeyword) Action() common.Action {
	return d.action
}

func NewDomainKeyword(rule *config.Rule) *DomainKeyword {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &DomainKeyword{
		action:        action,
		domainKeyword: rule.MatchValue,
	}
}
