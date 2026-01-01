package match

import (
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
	req := metadata.Request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	url := scheme + "://" + req.Host + req.URL.RequestURI()
	slog.Info("URLRegex Match", slog.String("url", url))
	match, _ := h.regex.MatchString(url)
	return match
}

func (h *URLRegex) Action() common.Action {
	return h.action
}

func (h *URLRegex) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(h.Type())),
		slog.String("url_regex", h.regex.String()),
		slog.Any("action", h.action),
	)
}

func NewURLRegex(rule *config.Rule, recorder *statistics.Recorder) *URLRegex {
	action := action.NewAction(rule, recorder)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	regex, err := regexp2.Compile(rule.MatchValue, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		regex = nil
	}

	return &URLRegex{
		action: action,
		regex:  regex,
	}
}
