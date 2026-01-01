package common

type ActionType string

const (
	ActionReplace      ActionType = "REPLACE"
	ActionReplaceRegex ActionType = "REPLACE-REGEX"
	ActionDelete       ActionType = "DELETE"
	ActionAdd          ActionType = "ADD"
	ActionDirect       ActionType = "DIRECT"
	ActionDrop         ActionType = "DROP"
)

type Action interface {
	Type() ActionType
	Execute(metadata *Metadata) (bool, error)
}
