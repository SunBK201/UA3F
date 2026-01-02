package match

import (
	"log/slog"
	"strings"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type HeaderKeyword struct {
	action  common.Action
	header  string
	keyword string
}

func (h *HeaderKeyword) Type() common.RuleType {
	return common.RuleTypeHeaderKeyword
}
func (h *HeaderKeyword) Match(metadata *common.Metadata) bool {
	header := metadata.Request.Header.Get(h.header)
	return strings.Contains(strings.ToLower(header), strings.ToLower(h.keyword))
}

func (h *HeaderKeyword) Action() common.Action {
	return h.action
}

func (h *HeaderKeyword) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(h.Type())),
		slog.String("header", h.header),
		slog.String("keyword", h.keyword),
		slog.Any("action", h.action),
	)
}

func NewHeaderKeyword(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *HeaderKeyword {
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

	return &HeaderKeyword{
		action:  a,
		header:  rule.MatchHeader,
		keyword: rule.MatchValue,
	}
}
