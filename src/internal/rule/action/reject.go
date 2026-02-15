package action

import (
	"encoding/json"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Reject struct {
	recorder  *statistics.Recorder
	direction common.Direction
}

func (d *Reject) Type() common.ActionType {
	return common.ActionReject
}

func (d *Reject) Execute(metadata *common.Metadata) (bool, error) {
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Reject Request")
	return false, nil
}

func (d *Reject) Direction() common.Direction {
	return d.direction
}

func (d *Reject) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":      d.Type(),
		"direction": d.direction,
	})
}

func (d *Reject) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.Any("direction", d.direction),
	)
}

func NewReject(recorder *statistics.Recorder, direction common.Direction) *Reject {
	return &Reject{
		recorder:  recorder,
		direction: direction,
	}
}
