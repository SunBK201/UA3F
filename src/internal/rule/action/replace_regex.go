package action

import (
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type ReplaceRegex struct {
	recorder      *statistics.Recorder
	replaceRegex  *regexp2.Regexp
	replaceHeader string
	replaceValue  string
	contine       bool
}

func (r *ReplaceRegex) Type() common.ActionType {
	return common.ActionReplaceRegex
}

func (r *ReplaceRegex) Execute(metadata *common.Metadata) (bool, error) {
	header := metadata.Request.Header.Get(r.replaceHeader)

	if header == "" {
		return r.contine, nil
	}

	replaceValue, err := r.replaceRegex.Replace(header, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("r.replaceRegex.Replace", "error", err)
		replaceValue = r.replaceValue
	}

	metadata.Request.Header.Set(r.replaceHeader, replaceValue)

	if r.recorder != nil {
		r.recorder.AddRecord(&statistics.RewriteRecord{
			Host:       metadata.DestAddr(),
			OriginalUA: header,
			MockedUA:   replaceValue,
		})
	}
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.replaceHeader, header, replaceValue))

	return r.contine, nil
}

func (r *ReplaceRegex) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("header", r.replaceHeader),
		slog.String("regex", r.replaceRegex.String()),
		slog.String("value", r.replaceValue),
		slog.Bool("continue", r.contine),
	)
}

func NewReplaceRegex(recorder *statistics.Recorder, replaceHeader, replaceRegex string, replaceValue string, contine bool) *ReplaceRegex {
	regex, err := regexp2.Compile("(?i)"+replaceRegex, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		return nil
	}

	return &ReplaceRegex{
		recorder:      recorder,
		replaceRegex:  regex,
		replaceHeader: replaceHeader,
		replaceValue:  replaceValue,
		contine:       contine,
	}
}
