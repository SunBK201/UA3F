package action

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/common"
)

type Replace struct {
	header string
	value  string
}

func (r *Replace) Type() common.ActionType {
	return common.ActionReplace
}

func (r *Replace) Execute(metadata *common.Metadata) (string, string) {
	header := metadata.Request.Header.Get(r.header)
	metadata.Request.Header.Set(r.header, r.value)
	log.LogInfoWithAddr(metadata.SrcAddr, metadata.DestAddr, fmt.Sprintf("Rewrite %s from (%s) to (%s)", r.header, header, r.value))
	return header, r.value
}

func (r *Replace) Header() string {
	return r.header
}

func NewReplace(header, value string) *Replace {
	return &Replace{
		header: header,
		value:  value,
	}
}
