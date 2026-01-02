package header

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Replace struct {
	recorder  *statistics.Recorder
	header    string
	value     string
	contine   bool
	direction common.Direction
}

func (r *Replace) Type() common.ActionType {
	return common.ActionReplace
}

func (r *Replace) Execute(metadata *common.Metadata) (bool, error) {
	var header string
	switch r.direction {
	case common.DirectionRequest:
		if metadata.Request == nil {
			return r.contine, fmt.Errorf("Request is nil")
		}
		header = metadata.Request.Header.Get(r.header)
		metadata.Request.Header.Set(r.header, r.value)
	case common.DirectionResponse:
		if metadata.Response == nil {
			return r.contine, fmt.Errorf("Response is nil")
		}
		header = metadata.Response.Header.Get(r.header)
		metadata.Response.Header.Set(r.header, r.value)
	case common.DirectionDual:
	default:
		return r.contine, fmt.Errorf("Unknown direction %s", r.direction)
	}

	if header == "" {
		return r.contine, nil
	}

	if r.recorder != nil {
		r.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: header,
			MockedUA:   r.value,
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.header, header, r.value))

	return r.contine, nil
}

func (r *Replace) Direction() common.Direction {
	return r.direction
}

func (r *Replace) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("header", r.header),
		slog.String("value", r.value),
		slog.Bool("continue", r.contine),
		slog.String("direction", string(r.direction)),
	)
}

func NewReplace(recorder *statistics.Recorder, header, value string, contine bool, direction common.Direction) *Replace {
	return &Replace{
		recorder:  recorder,
		header:    header,
		value:     value,
		contine:   contine,
		direction: direction,
	}
}
