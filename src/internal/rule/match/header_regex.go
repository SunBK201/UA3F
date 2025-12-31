package match

import (
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

type HeaderRegex struct {
	action common.Action
	header string
	regex  *regexp2.Regexp
}

func (h *HeaderRegex) Type() common.RuleType {
	return common.RuleTypeHeaderRegex
}
func (h *HeaderRegex) Match(metadata *common.Metadata) bool {
	if h.regex == nil {
		return false
	}
	header := metadata.Request.Header.Get(h.header)
	match, _ := h.regex.MatchString(header)
	return match
}

func (h *HeaderRegex) Action() common.Action {
	return h.action
}

func NewHeaderRegex(rule *config.Rule) *HeaderRegex {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	regex, err := regexp2.Compile("(?i)"+rule.MatchValue, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		regex = nil
	}

	return &HeaderRegex{
		action: action,
		header: rule.MatchHeader,
		regex:  regex,
	}
}
