package match

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type DestPort struct {
	action common.Action
	port   string
}

func (d *DestPort) Type() common.RuleType {
	return common.RuleTypeDestPort
}

func (d *DestPort) Match(metadata *common.Metadata) bool {
	return metadata.ConnLink.RPort() == d.port
}

func (d *DestPort) Action() common.Action {
	return d.action
}

func (d *DestPort) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("port", d.port),
		slog.Any("action", d.action),
	)
}

func NewDestPort(rule *config.Rule, recorder *statistics.Recorder) *DestPort {
	action := action.NewAction(rule, recorder)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	return &DestPort{
		action: action,
		port:   rule.MatchValue,
	}
}
