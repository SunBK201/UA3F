package rewrite

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

type DirectRewriter struct {
}

func (r *DirectRewriter) RewriteRequest(metadata *common.Metadata) (decision *common.RewriteDecision) {
	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	decision = &common.RewriteDecision{
		Action: action.DirectAction,
	}
	_, err := decision.Action.Execute(metadata)
	if err != nil {
		log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
	}
	return decision
}

func (r *DirectRewriter) RewriteResponse(metadata *common.Metadata) (decision *common.RewriteDecision) {
	return &common.RewriteDecision{
		Action: action.DirectAction,
	}
}

func (r *DirectRewriter) ServeRequest() bool {
	return false
}

func (r *DirectRewriter) ServeResponse() bool {
	return false
}

func (r *DirectRewriter) HeaderRules() []common.Rule {
	return nil
}

func (r *DirectRewriter) BodyRules() []common.Rule {
	return nil
}

func (r *DirectRewriter) RedirectRules() []common.Rule {
	return nil
}

func NewDirectRewriter() *DirectRewriter {
	return &DirectRewriter{}
}
