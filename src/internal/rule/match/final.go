package match

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type final struct {
	action common.Action
}

func (f *final) Type() common.RuleType {
	return common.RuleTypeFinal
}

func (f *final) Match(meta *common.Metadata) bool {
	return true
}

func (f *final) Action() common.Action {
	return f.action
}

func (f *final) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(f.Type())),
		slog.Any("action", f.action),
	)
}

func NewFinal(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *final {
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

	return &final{
		action: a,
	}
}
