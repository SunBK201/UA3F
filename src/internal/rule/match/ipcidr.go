package match

import (
	"log/slog"
	"net"
	"strings"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

type IPCIDR struct {
	action common.Action
	ipNet  *net.IPNet
}

func (i *IPCIDR) Type() common.RuleType {
	return common.RuleTypeIPCIDR
}

func (i *IPCIDR) Match(metadata *common.Metadata) bool {
	if i.ipNet == nil {
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

func NewIPCIDR(rule *config.Rule) *IPCIDR {
	action := action.NewAction(rule)
	if action == nil {
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
		action: action,
		ipNet:  ipNet,
	}
}
