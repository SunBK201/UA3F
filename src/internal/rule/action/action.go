package action

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/statistics"
)

var (
	DirectAction       = NewDirect(nil)
	DropRequestAction  = NewDrop(nil, common.DirectionRequest)
	DropResponseAction = NewDrop(nil, common.DirectionResponse)
)

func InitActions(recorder *statistics.Recorder) {
	DirectAction.SetRecorder(recorder)
}

func NewAction(rule *config.Rule, recorder *statistics.Recorder) common.Action {
	var direction common.Direction
	if rule.RewriteDirection == "" {
		direction = common.DirectionRequest
	} else {
		direction = common.Direction(rule.RewriteDirection)
	}

	switch common.ActionType(rule.Action) {
	case common.ActionDirect:
		DirectAction.SetRecorder(recorder)
		return DirectAction
	case common.ActionDrop:
		switch common.Direction(rule.RewriteDirection) {
		case common.DirectionRequest:
			return DropRequestAction
		case common.DirectionResponse:
			return DropResponseAction
		default:
			return nil
		}
	case common.ActionDelete:
		return NewDelete(recorder, rule.RewriteHeader, rule.Continue, direction)
	case common.ActionAdd:
		return NewAdd(recorder, rule.RewriteHeader, rule.RewriteValue, rule.Continue, direction)
	case common.ActionReplace:
		return NewReplace(recorder, rule.RewriteHeader, rule.RewriteValue, rule.Continue, direction)
	case common.ActionReplaceRegex:
		return NewReplaceRegex(recorder, rule.RewriteHeader, rule.RewriteRegex, rule.RewriteValue, rule.Continue, direction)
	default:
		return nil
	}
}
