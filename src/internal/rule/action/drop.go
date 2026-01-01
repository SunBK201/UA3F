package action

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Drop struct {
	recorder *statistics.Recorder
}

func (d *Drop) Type() common.ActionType {
	return common.ActionDrop
}

func (d *Drop) Execute(metadata *common.Metadata) (bool, error) {
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Drop Request")
	return false, nil
}

func (d *Drop) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
	)
}

func NewDrop(recorder *statistics.Recorder) *Drop {
	return &Drop{
		recorder: recorder,
	}
}
