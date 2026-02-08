package header

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Delete struct {
	recorder  *statistics.Recorder
	header    string
	contine   bool
	direction common.Direction
}

func (d *Delete) Type() common.ActionType {
	return common.ActionDelete
}

func (d *Delete) Execute(metadata *common.Metadata) (bool, error) {
	var header string
	switch d.direction {
	case common.DirectionRequest:
		if metadata.Request == nil {
			return d.contine, fmt.Errorf("request is nil")
		}
		header = metadata.Request.Header.Get(d.header)
		metadata.Request.Header.Del(d.header)
	case common.DirectionResponse:
		if metadata.Response == nil {
			return d.contine, fmt.Errorf("response is nil")
		}
		header = metadata.Response.Header.Get(d.header)
		metadata.Response.Header.Del(d.header)
	case common.DirectionDual:
	default:
		return d.contine, fmt.Errorf("unknown direction %s", d.direction)
	}

	if header == "" {
		return d.contine, nil
	}

	if d.recorder != nil {
		d.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: header,
			MockedUA:   "",
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Delete Header %s (%s)", d.header, header))

	return d.contine, nil
}

func (d *Delete) Direction() common.Direction {
	return d.direction
}

func (d *Delete) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":      d.Type(),
		"header":    d.header,
		"continue":  d.contine,
		"direction": d.direction,
	})
}

func (d *Delete) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("header", d.header),
		slog.Bool("continue", d.contine),
		slog.String("direction", string(d.direction)),
	)
}

func NewDelete(recorder *statistics.Recorder, header string, contine bool, direction common.Direction) *Delete {
	return &Delete{
		recorder:  recorder,
		header:    header,
		contine:   contine,
		direction: direction,
	}
}
