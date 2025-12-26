package match

import (
	"log/slog"
	"net"
	"strconv"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

type DestPort struct {
	action common.Action
	port   uint16
}

func (d *DestPort) Type() common.RuleType {
	return common.RuleTypeDestPort
}

func (d *DestPort) Match(meta *common.Metadata) bool {
	_, portStr, err := net.SplitHostPort(meta.DestAddr)
	if err != nil {
		return false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}
	return uint16(port) == d.port
}

func (d *DestPort) Action() common.Action {
	return d.action
}

func NewDestPort(rule *config.Rule) *DestPort {
	action := action.NewAction(rule)
	if action == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	port64, err := strconv.ParseUint(rule.MatchValue, 10, 16)
	if err != nil {
		slog.Error("strconv.ParseUint", "error", err)
		return nil
	}
	port := uint16(port64)

	return &DestPort{
		action: action,
		port:   port,
	}
}
