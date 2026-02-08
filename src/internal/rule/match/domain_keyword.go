package match

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
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

func (d *DomainKeyword) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":           d.Type(),
		"domain_keyword": d.domainKeyword,
		"action":         d.action,
	})
}

func (d *DomainKeyword) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("domain_keyword", d.domainKeyword),
		slog.Any("action", d.action),
	)
}

func NewDomainKeyword(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *DomainKeyword {
	var a common.Action
	switch target {
	case common.ActionTargetHeader:
		a = action.NewHeaderAction(rule, recorder)
	case common.ActionTargetBody:
		a = action.NewBodyAction(rule, recorder)
	case common.ActionTargetURL:
		a = action.NewURLAction(rule, recorder)
	default:
		slog.Error("unknown target", "target", target)
		return nil
	}
	if a == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &DomainKeyword{
		action:        a,
		domainKeyword: rule.MatchValue,
	}
}
