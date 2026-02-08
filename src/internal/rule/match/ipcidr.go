package match

import (
	"encoding/json"
	"log/slog"
	"net"
	"strings"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type IPCIDR struct {
	action common.Action
	ipNet  *net.IPNet
}

func (i *IPCIDR) Type() common.RuleType {
	return common.RuleTypeIPCIDR
}

func (i *IPCIDR) Match(metadata *common.Metadata) bool {
	if i.ipNet == nil || metadata.ConnLink == nil {
		return false
	}
	ip := net.ParseIP(metadata.ConnLink.RIP())
	if ip == nil {
		return false
	}
	return i.ipNet.Contains(ip)
}

func (i *IPCIDR) Action() common.Action {
	return i.action
}

func (i *IPCIDR) MarshalJSON() ([]byte, error) {
	var cidr string
	if i.ipNet != nil {
		cidr = i.ipNet.String()
	}
	return json.Marshal(map[string]any{
		"type":    i.Type(),
		"ip_cidr": cidr,
		"action":  i.action,
	})
}

func (i *IPCIDR) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(i.Type())),
		slog.String("ip_cidr", i.ipNet.String()),
		slog.Any("action", i.action),
	)
}

func NewIPCIDR(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *IPCIDR {
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

	if !strings.Contains(rule.MatchValue, "/") {
		rule.MatchValue += "/32"
	}

	_, ipNet, err := net.ParseCIDR(rule.MatchValue)
	if err != nil {
		slog.Error("net.ParseCIDR", "error", err)
		return nil
	}

	return &IPCIDR{
		action: a,
		ipNet:  ipNet,
	}
}
