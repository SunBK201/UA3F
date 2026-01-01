package action

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/statistics"
)

var (
	DirectAction = NewDirect(nil)
	DropAction   = NewDrop(nil)
)

func InitActions(recorder *statistics.Recorder) {
	DirectAction.SetRecorder(recorder)
}

func NewAction(rule *config.Rule, recorder *statistics.Recorder) common.Action {
	switch common.ActionType(rule.Action) {
	case common.ActionDirect:
		DirectAction.SetRecorder(recorder)
		return DirectAction
	case common.ActionDrop:
		return DropAction
	case common.ActionDelete:
		return NewDelete(recorder, rule.RewriteHeader, rule.Continue)
	case common.ActionReplace:
		return NewReplace(recorder, rule.RewriteHeader, rule.RewriteValue, rule.Continue)
	case common.ActionReplaceRegex:
		return NewReplaceRegex(recorder, rule.RewriteHeader, rule.RewriteRegex, rule.RewriteValue, rule.Continue)
	default:
		return nil
	}
}
