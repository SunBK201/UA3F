package common

type ActionType string

const (
	ActionReplace        ActionType = "REPLACE"
	ActionReplaceRegex   ActionType = "REPLACE-REGEX"
	ActionDelete         ActionType = "DELETE"
	ActionAdd            ActionType = "ADD"
	ActionDirect         ActionType = "DIRECT"
	ActionDrop           ActionType = "DROP"
	ActionRedirect302    ActionType = "REDIRECT-302"
	ActionRedirect307    ActionType = "REDIRECT-307"
	ActionRedirectHeader ActionType = "REDIRECT-HEADER"
)

type ActionTarget string

const (
	ActionTargetHeader ActionTarget = "HEADER"
	ActionTargetBody   ActionTarget = "BODY"
	ActionTargetURL    ActionTarget = "URL"
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
