package rewrite

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

type DirectRewriter struct {
}

func (r *DirectRewriter) RewriteRequest(metadata *common.Metadata) (decision *RewriteDecision) {
	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}
	_, err := decision.Action.Execute(metadata)
	if err != nil {
		log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
	}
	return decision
}

func (r *DirectRewriter) RewriteResponse(metadata *common.Metadata) (decision *RewriteDecision) {
	return &RewriteDecision{
		Action: action.DirectAction,
	}
}

func (r *DirectRewriter) ServeRequest() bool {
	return false
}

func (r *DirectRewriter) ServeResponse() bool {
	return false
}

func NewDirectRewriter() *DirectRewriter {
	return &DirectRewriter{}
}
