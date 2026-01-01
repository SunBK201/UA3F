package action

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Add struct {
	recorder *statistics.Recorder
	header   string
	value    string
	contine  bool
}

func (a *Add) Type() common.ActionType {
	return common.ActionAdd
}

func (a *Add) Execute(metadata *common.Metadata) (bool, error) {
	metadata.Request.Header.Add(a.header, a.value)

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

func (a *Add) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(a.Type())),
		slog.String("header", a.header),
		slog.String("value", a.value),
		slog.Bool("continue", a.contine),
	)
}

func NewAdd(recorder *statistics.Recorder, header string, value string, contine bool) *Add {
	return &Add{
		recorder: recorder,
		header:   header,
		value:    value,
		contine:  contine,
	}
}
