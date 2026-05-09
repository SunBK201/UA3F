package match

import (
	"encoding/json"
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
	return metadata.DestPort() == d.port
}

func (d *DestPort) Action() common.Action {
	return d.action
}

func (d *DestPort) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":   d.Type(),
		"port":   d.port,
		"action": d.action,
	})
}

func (d *DestPort) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("port", d.port),
		slog.Any("action", d.action),
	)
}

func NewDestPort(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *DestPort {
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

	return &DestPort{
		action: a,
		port:   rule.MatchValue,
	}
}
