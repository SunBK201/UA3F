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

func (d *DomainSuffix) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":          d.Type(),
		"domain_suffix": d.domainSuffix,
		"action":        d.action,
	})
}

func (d *DomainSuffix) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("domain_suffix", d.domainSuffix),
		slog.Any("action", d.action),
	)
}

func NewDomainSuffix(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *DomainSuffix {
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

	return &DomainSuffix{
		action:       a,
		domainSuffix: rule.MatchValue,
	}
}
