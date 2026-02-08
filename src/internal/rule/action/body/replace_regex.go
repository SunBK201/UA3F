package body

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type ReplaceRegex struct {
	recorder     *statistics.Recorder
	replaceRegex *regexp2.Regexp
	replaceValue string
	contine      bool
	direction    common.Direction
}

func (r *ReplaceRegex) Type() common.ActionType {
	return common.ActionReplaceRegex
}

func (r *ReplaceRegex) Execute(metadata *common.Metadata) (bool, error) {
	var body []byte
	switch r.direction {
	case common.DirectionRequest:
		if metadata.Request == nil {
			return r.contine, fmt.Errorf("request is nil")
		}
		body = metadata.RequestBody(true)
	case common.DirectionResponse:
		if metadata.Response == nil {
			return r.contine, fmt.Errorf("response is nil")
		}
		body = metadata.ResponseBody(true)
	case common.DirectionDual:
	default:
		return r.contine, fmt.Errorf("unknown direction %s", r.direction)
	}

	bodyStr := string(body)

	replaceValue, err := r.replaceRegex.Replace(bodyStr, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("r.replaceRegex.Replace", "error", err)
		replaceValue = bodyStr
	}

	switch r.direction {
	case common.DirectionRequest:
		if metadata.Request == nil {
			return r.contine, fmt.Errorf("request is nil")
		}
		metadata.UpdateRequestBody([]byte(replaceValue), true)
	case common.DirectionResponse:
		if metadata.Response == nil {
			return r.contine, fmt.Errorf("response is nil")
		}
		metadata.UpdateResponseBody([]byte(replaceValue), true)
	}

	return r.contine, nil
}

func (r *ReplaceRegex) Direction() common.Direction {
	return r.direction
}

func (r *ReplaceRegex) MarshalJSON() ([]byte, error) {
	var regex string
	if r.replaceRegex != nil {
		regex = r.replaceRegex.String()
	}
	return json.Marshal(map[string]any{
		"type":      r.Type(),
		"regex":     regex,
		"value":     r.replaceValue,
		"continue":  r.contine,
		"direction": r.direction,
	})
}

func (r *ReplaceRegex) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("regex", r.replaceRegex.String()),
		slog.String("value", r.replaceValue),
		slog.Bool("continue", r.contine),
		slog.String("direction", string(r.direction)),
	)
}

func NewReplaceRegex(recorder *statistics.Recorder, replaceRegex string, replaceValue string, contine bool, direction common.Direction) *ReplaceRegex {
	regex, err := regexp2.Compile(replaceRegex, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		return nil
	}

	return &ReplaceRegex{
		recorder:     recorder,
		replaceRegex: regex,
		replaceValue: replaceValue,
		contine:      contine,
		direction:    direction,
	}
}
