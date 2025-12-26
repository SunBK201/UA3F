package action

import "github.com/sunbk201/ua3f/internal/rule/common"

type Drop struct{}

func (d *Drop) Type() common.ActionType {
	return common.ActionDrop
}

func (d *Drop) Execute(metadata *common.Metadata) (string, string) {
	return "", ""
}

func (d *Drop) Header() string {
	return ""
}

func NewDrop() *Drop {
	return &Drop{}
}
