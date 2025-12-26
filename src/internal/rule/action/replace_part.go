package action

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

type ReplacePart struct {
	matchValue    string
	matchRegex    *regexp2.Regexp
	replaceHeader string
	replaceValue  string
}

func (r *ReplacePart) Type() common.ActionType {
	return common.ActionReplacePart
}

func (r *ReplacePart) Execute(metadata *common.Metadata) (string, string) {
	var err error
	var replaceValue string

	header := metadata.Request.Header.Get(r.replaceHeader)

	if r.matchRegex != nil {
		replaceValue, err = r.matchRegex.Replace(header, r.replaceValue, -1, -1)
		if err != nil {
			slog.Error("r.matchRegex.Replace", "error", err)
			replaceValue = r.replaceValue
		}
	} else {
		replaceValue = strings.ReplaceAll(header, r.matchValue, r.replaceValue)
	}

	log.LogInfoWithAddr(metadata.SrcAddr, metadata.DestAddr, fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.replaceHeader, header, replaceValue))
	metadata.Request.Header.Set(r.replaceHeader, replaceValue)
	return header, replaceValue
}

func (r *ReplacePart) Header() string {
	return r.replaceHeader
}

func NewReplacePart(matchValue string, replaceHeader, replaceValue string, regex bool) *ReplacePart {
	var err error
	var matchRegex *regexp2.Regexp

	if regex {
		matchRegex, err = regexp2.Compile("(?i)"+matchValue, regexp2.None)
		if err != nil {
			slog.Error("regexp2.Compile", "error", err)
			return nil
		}
	}

	return &ReplacePart{
		matchValue:    matchValue,
		matchRegex:    matchRegex,
		replaceHeader: replaceHeader,
		replaceValue:  replaceValue,
	}
}
