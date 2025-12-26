package match

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
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

func NewFinal(rule *config.Rule) *final {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &final{
		action: action,
	}
}
