package action

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
)

type Delete struct {
	header string
}

func (d *Delete) Type() common.ActionType {
	return common.ActionDelete
}

func (d *Delete) Execute(metadata *common.Metadata) (string, string) {
	header := metadata.Request.Header.Get(d.header)
	metadata.Request.Header.Set(d.header, "")
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Delete Header %s (%s)", d.header, header))
	return header, ""
}

func (d *Delete) Header() string {
	return d.header
}

func NewDelete(header string) *Delete {
	return &Delete{
		header: header,
	}
}
