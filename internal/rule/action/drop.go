package action

import (
	"encoding/json"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Drop struct {
	recorder  *statistics.Recorder
	direction common.Direction
}

func (d *Drop) Type() common.ActionType {
	return common.ActionDrop
}

func (d *Drop) Execute(metadata *common.Metadata) (bool, error) {
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Drop Request")
	return false, nil
}

func (d *Drop) Direction() common.Direction {
	return d.direction
}

func (d *Drop) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":      d.Type(),
		"direction": d.direction,
	})
}

func (d *Drop) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.Any("direction", d.direction),
	)
}

func NewDrop(recorder *statistics.Recorder, direction common.Direction) *Drop {
	return &Drop{
		recorder:  recorder,
		direction: direction,
	}
}
