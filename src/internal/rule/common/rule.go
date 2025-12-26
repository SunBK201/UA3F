package common

type RuleType string

const (
	RuleTypeHeaderKeyword RuleType = "HEADER-KEYWORD"
	RuleTypeHeaderRegex   RuleType = "HEADER-REGEX"
	RuleTypeIPCIDR        RuleType = "IP-CIDR"
	RuleTypeSrcIP         RuleType = "SRC-IP"
	RuleTypeDestPort      RuleType = "DEST-PORT"
	RuleTypeFinal         RuleType = "FINAL"
)

type Rule interface {
	Type() RuleType
	Match(metadata *Metadata) bool
	Action() Action
}
