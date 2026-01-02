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

type ActionTarget string

const (
	ActionTargetHeader ActionTarget = "HEADER"
	ActionTargetBody   ActionTarget = "BODY"
)

type Direction string

const (
	DirectionDual     Direction = "DUAL"
	DirectionRequest  Direction = "REQUEST"
	DirectionResponse Direction = "RESPONSE"
)

type Action interface {
	Type() ActionType
	Execute(metadata *Metadata) (bool, error)
	Direction() Direction
}
