package action

import "github.com/sunbk201/ua3f/internal/common"

type Direct struct{}

func (d *Direct) Type() common.ActionType {
	return common.ActionDirect
}

func (d *Direct) Execute(metadata *common.Metadata) (string, string) {
	return "", ""
}

func (d *Direct) Header() string {
	return ""
}

func NewDirect() *Direct {
	return &Direct{}
}
