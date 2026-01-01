package action

import (
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Direct struct {
	recorder *statistics.Recorder
}

func (d *Direct) Type() common.ActionType {
	return common.ActionDirect
}

func (d *Direct) Execute(metadata *common.Metadata) error {
	ua := metadata.UserAgent()
	if ua == "" {
		return nil
	}
	if d.recorder != nil {
		d.recorder.AddRecord(&statistics.PassThroughRecord{
			SrcAddr:  metadata.SrcAddr(),
			DestAddr: metadata.DestAddr(),
			UA:       ua,
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), "Direct Forwarding with User-Agent: "+ua)
	return nil
}

func (d *Direct) SetRecorder(recorder *statistics.Recorder) {
	d.recorder = recorder
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
