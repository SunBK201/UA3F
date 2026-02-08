package action

import (
	"encoding/json"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Direct struct {
	recorder *statistics.Recorder
}

func (d *Direct) Type() common.ActionType {
	return common.ActionDirect
}

func (d *Direct) Execute(metadata *common.Metadata) (bool, error) {
	ua := metadata.UserAgent()
	if ua == "" {
		return false, nil
	}
	return false, nil
}

func (d *Direct) SetRecorder(recorder *statistics.Recorder) {
	d.recorder = recorder
}

func (d *Direct) Direction() common.Direction {
	return common.DirectionDual
}

func (d *Direct) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": d.Type(),
	})
}

func (d *Direct) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
	)
}

func NewDirect(recorder *statistics.Recorder) *Direct {
	return &Direct{
		recorder: recorder,
	}
}
