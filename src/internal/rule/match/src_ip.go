package match

import (
	"log/slog"
	"net"
	"strings"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type SrcIP struct {
	action common.Action
	ipNet  *net.IPNet
}

func (s *SrcIP) Type() common.RuleType {
	return common.RuleTypeSrcIP
}

func (s *SrcIP) Match(metadata *common.Metadata) bool {
	if s.ipNet == nil || metadata.ConnLink == nil {
		return false
	}
	ip := net.ParseIP(metadata.ConnLink.LIP())
	if ip == nil {
		return false
	}
	return s.ipNet.Contains(ip)
}

func (s *SrcIP) Action() common.Action {
	return s.action
}

func (s *SrcIP) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(s.Type())),
		slog.String("ip_cidr", s.ipNet.String()),
		slog.Any("action", s.action),
	)
}

func NewSrcIP(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *SrcIP {
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

	if !strings.Contains(rule.MatchValue, "/") {
		rule.MatchValue += "/32"
	}

	_, ipNet, err := net.ParseCIDR(rule.MatchValue)
	if err != nil {
		slog.Error("net.ParseCIDR", "error", err)
		return nil
	}

	return &SrcIP{
		action: a,
		ipNet:  ipNet,
	}
}
