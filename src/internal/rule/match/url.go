package match

import (
	"encoding/json"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type URLRegex struct {
	action common.Action
	regex  *regexp2.Regexp
}

func (h *URLRegex) Type() common.RuleType {
	return common.RuleTypeURLRegex
}
func (h *URLRegex) Match(metadata *common.Metadata) bool {
	if h.regex == nil {
		return false
	}
	match, _ := h.regex.MatchString(metadata.URL())
	return match
}

func (h *URLRegex) Action() common.Action {
	return h.action
}

func (h *URLRegex) MarshalJSON() ([]byte, error) {
	var regex string
	if h.regex != nil {
		regex = h.regex.String()
	}
	return json.Marshal(map[string]any{
		"type":      h.Type(),
		"url_regex": regex,
		"action":    h.action,
	})
}

func (h *URLRegex) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(h.Type())),
		slog.String("url_regex", h.regex.String()),
		slog.Any("action", h.action),
	)
}

func NewURLRegex(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *URLRegex {
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

	regex, err := regexp2.Compile(rule.MatchValue, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		regex = nil
	}

	return &URLRegex{
		action: a,
		regex:  regex,
	}
}
