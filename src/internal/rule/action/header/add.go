package header

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Add struct {
	recorder  *statistics.Recorder
	header    string
	value     string
	direction common.Direction
	contine   bool
}

func (a *Add) Type() common.ActionType {
	return common.ActionAdd
}

func (a *Add) Execute(metadata *common.Metadata) (bool, error) {
	switch a.direction {
	case common.DirectionRequest:
		if metadata.Request == nil {
			return a.contine, fmt.Errorf("request is nil")
		}
		metadata.Request.Header.Add(a.header, a.value)
	case common.DirectionResponse:
		if metadata.Response == nil {
			return a.contine, fmt.Errorf("response is nil")
		}
		metadata.Response.Header.Add(a.header, a.value)
	case common.DirectionDual:
	default:
		return a.contine, fmt.Errorf("unknown direction %s", a.direction)
	}

	if a.recorder != nil {
		a.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: "",
			MockedUA:   a.value,
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Add Header %s (%s)", a.header, a.value))

	return a.contine, nil
}

func (a *Add) Direction() common.Direction {
	return a.direction
}

func (a *Add) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(a.Type())),
		slog.String("header", a.header),
		slog.String("value", a.value),
		slog.Bool("continue", a.contine),
		slog.String("direction", string(a.direction)),
	)
}

func NewAdd(recorder *statistics.Recorder, header string, value string, contine bool, direction common.Direction) *Add {
	return &Add{
		recorder:  recorder,
		header:    header,
		value:     value,
		contine:   contine,
		direction: direction,
	}
}
