package action

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
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
	case common.ActionReplaceRegex:
		return NewReplaceRegex(rule.RewriteHeader, rule.RewriteRegex, rule.RewriteValue)
	default:
		return nil
	}
}
