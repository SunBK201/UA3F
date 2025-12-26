package match

import (
	"log/slog"
	"strings"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
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

func NewHeaderKeyword(rule *config.Rule) *HeaderKeyword {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &HeaderKeyword{
		action:  action,
		header:  rule.MatchHeader,
		keyword: rule.MatchValue,
	}
}
