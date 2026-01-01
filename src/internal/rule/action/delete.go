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
	contine  bool
}

func (d *Delete) Type() common.ActionType {
	return common.ActionDelete
}

func (d *Delete) Execute(metadata *common.Metadata) (bool, error) {
	header := metadata.Request.Header.Get(d.header)

	if header == "" {
		return d.contine, nil
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
	return d.contine, nil
}

func (d *Delete) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.String("header", d.header),
		slog.Bool("continue", d.contine),
	)
}

func NewDelete(recorder *statistics.Recorder, header string, contine bool) *Delete {
	return &Delete{
		recorder: recorder,
		header:   header,
		contine:  contine,
	}
}
