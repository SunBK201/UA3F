package action

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action/body"
	"github.com/sunbk201/ua3f/internal/rule/action/header"
	"github.com/sunbk201/ua3f/internal/rule/action/redirect"
	"github.com/sunbk201/ua3f/internal/statistics"
)

var (
	DirectAction         = NewDirect(nil)
	DropRequestAction    = NewDrop(nil, common.DirectionRequest)
	DropResponseAction   = NewDrop(nil, common.DirectionResponse)
	RejectRequestAction  = NewReject(nil, common.DirectionRequest)
	RejectResponseAction = NewReject(nil, common.DirectionResponse)
)

func InitActions(recorder *statistics.Recorder) {
	DirectAction.SetRecorder(recorder)
}

func NewHeaderAction(rule *config.Rule, recorder *statistics.Recorder) common.Action {
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
	case common.ActionReject:
		switch common.Direction(rule.RewriteDirection) {
		case common.DirectionRequest:
			return RejectRequestAction
		case common.DirectionResponse:
			return RejectResponseAction
		default:
			return nil
		}
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
		return header.NewDelete(recorder, rule.RewriteHeader, rule.Continue, direction)
	case common.ActionAdd:
		return header.NewAdd(recorder, rule.RewriteHeader, rule.RewriteValue, rule.Continue, direction)
	case common.ActionReplace:
		return header.NewReplace(recorder, rule.RewriteHeader, rule.RewriteValue, rule.Continue, direction)
	case common.ActionReplaceRegex:
		return header.NewReplaceRegex(recorder, rule.RewriteHeader, rule.RewriteRegex, rule.RewriteValue, rule.Continue, direction)
	default:
		return nil
	}
}

func NewBodyAction(rule *config.Rule, recorder *statistics.Recorder) common.Action {
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
	case common.ActionReject:
		switch common.Direction(rule.RewriteDirection) {
		case common.DirectionRequest:
			return RejectRequestAction
		case common.DirectionResponse:
			return RejectResponseAction
		default:
			return nil
		}
	case common.ActionDrop:
		switch common.Direction(rule.RewriteDirection) {
		case common.DirectionRequest:
			return DropRequestAction
		case common.DirectionResponse:
			return DropResponseAction
		default:
			return nil
		}
	case common.ActionReplaceRegex:
		return body.NewReplaceRegex(recorder, rule.RewriteRegex, rule.RewriteValue, rule.Continue, direction)
	default:
		return nil
	}
}

func NewURLAction(rule *config.Rule, recorder *statistics.Recorder) common.Action {
	switch common.ActionType(rule.Action) {
	case common.ActionDirect:
		DirectAction.SetRecorder(recorder)
		return DirectAction
	case common.ActionRedirect302:
		return redirect.NewRedirect302(rule.RewriteRegex, rule.RewriteValue)
	case common.ActionRedirect307:
		return redirect.NewRedirect307(rule.RewriteRegex, rule.RewriteValue)
	case common.ActionRedirectHeader:
		return redirect.NewRedirectHeader(rule.RewriteRegex, rule.RewriteValue)
	default:
		return nil
	}
}
