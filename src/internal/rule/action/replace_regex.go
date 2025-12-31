package action

import (
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
)

type ReplaceRegex struct {
	replaceRegex  *regexp2.Regexp
	replaceHeader string
	replaceValue  string
}

func (r *ReplaceRegex) Type() common.ActionType {
	return common.ActionReplaceRegex
}

func (r *ReplaceRegex) Execute(metadata *common.Metadata) (string, string) {
	header := metadata.Request.Header.Get(r.replaceHeader)

	replaceValue, err := r.replaceRegex.Replace(header, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("r.matchRegex.Replace", "error", err)
		replaceValue = r.replaceValue
	}

	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.replaceHeader, header, replaceValue))
	metadata.Request.Header.Set(r.replaceHeader, replaceValue)
	return header, replaceValue
}

func (r *ReplaceRegex) Header() string {
	return r.replaceHeader
}

func NewReplaceRegex(replaceHeader, replaceRegex string, replaceValue string) *ReplaceRegex {
	regex, err := regexp2.Compile("(?i)"+replaceRegex, regexp2.None)
	if err != nil {
		slog.Error("regexp2.Compile", "error", err)
		return nil
	}

	return &ReplaceRegex{
		replaceRegex:  regex,
		replaceHeader: replaceHeader,
		replaceValue:  replaceValue,
	}
}
