package action

import (
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

var (
	DirectAction = NewDirect()
	DropAction   = NewDrop()
)

func NewAction(rule *config.Rule) common.Action {
	switch common.ActionType(rule.Action) {
	case common.ActionDirect:
		return DirectAction
	case common.ActionDrop:
		return DropAction
	case common.ActionDelete:
		return NewDelete(rule.RewriteHeader)
	case common.ActionReplace:
		return NewReplace(rule.RewriteHeader, rule.RewriteValue)
	case common.ActionReplacePart:
		regex := rule.Type == string(common.RuleTypeHeaderRegex)
		return NewReplacePart(rule.MatchValue, rule.RewriteHeader, rule.RewriteValue, regex)
	default:
		return nil
	}
}
