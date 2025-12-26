package common

type ActionType string

const (
	ActionReplace     ActionType = "REPLACE"
	ActionReplacePart ActionType = "REPLACE-PART"
	ActionDelete      ActionType = "DELETE"
	ActionDirect      ActionType = "DIRECT"
	ActionDrop        ActionType = "DROP"
)

type Action interface {
	Type() ActionType
	Execute(metadata *Metadata) (string, string)
	Header() string
}
