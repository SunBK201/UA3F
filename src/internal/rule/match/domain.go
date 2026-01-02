package match

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
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

func (d *Domain) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("domain", d.domain),
		slog.Any("action", d.action),
	)
}

func NewDomain(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *Domain {
	var a common.Action
	switch target {
	case common.ActionTargetHeader:
		a = action.NewHeaderAction(rule, recorder)
	case common.ActionTargetBody:
		a = action.NewBodyAction(rule, recorder)
	default:
		slog.Error("unknown target", "target", target)
		return nil
	}
	if a == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &Domain{
		action: a,
		domain: rule.MatchValue,
	}
}
