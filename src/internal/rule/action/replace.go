package action

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Replace struct {
	recorder *statistics.Recorder
	header   string
	value    string
}

func (r *Replace) Type() common.ActionType {
	return common.ActionReplace
}

func (r *Replace) Execute(metadata *common.Metadata) error {
	header := metadata.Request.Header.Get(r.header)

	if header == "" {
		return nil
	}

	metadata.Request.Header.Set(r.header, r.value)

	if r.recorder != nil {
		r.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: header,
			MockedUA:   r.value,
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.header, header, r.value))
	return nil
}

func (r *Replace) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("header", r.header),
		slog.String("value", r.value),
	)
}

func NewReplace(recorder *statistics.Recorder, header, value string) *Replace {
	return &Replace{
		recorder: recorder,
		header:   header,
		value:    value,
	}
}
