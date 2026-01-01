package action

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Delete struct {
	recorder *statistics.Recorder
	header   string
}

func (d *Delete) Type() common.ActionType {
	return common.ActionDelete
}

func (d *Delete) Execute(metadata *common.Metadata) error {
	header := metadata.Request.Header.Get(d.header)

	if header == "" {
		return nil
	}

	metadata.Request.Header.Set(d.header, "")
	if d.recorder != nil {
		d.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: header,
			MockedUA:   "",
		})
	}

	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Delete Header %s (%s)", d.header, header))
	return nil
}

func (d *Delete) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("header", d.header),
	)
}

func NewDelete(recorder *statistics.Recorder, header string) *Delete {
	return &Delete{
		recorder: recorder,
		header:   header,
	}
}
