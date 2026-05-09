package common

type Rewriter interface {
	RewriteRequest(metadata *Metadata) (decision *RewriteDecision)
	RewriteResponse(metadata *Metadata) (decision *RewriteDecision)
	ServeRequest() bool
	ServeResponse() bool
	HeaderRules() []Rule
	BodyRules() []Rule
	RedirectRules() []Rule
}

type RewriteDecision struct {
	Action      Action
	MatchedRule Rule
	NeedCache   bool
	NeedSkip    bool
	Redirect    bool // URL Redirect

	Modified bool // NFQUEUE
	HasUA    bool // NFQUEUE
}
