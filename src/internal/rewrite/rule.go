package rewrite

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type RuleRewriter struct {
	RuleEngine *rule.Engine
	Recorder   *statistics.Recorder
}

func (r *RuleRewriter) RewriteRequest(metadata *common.Metadata) (decision *RewriteDecision) {
	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}

	var matchedRule common.Rule
	index := -1
	for {
		matchedRule, index = r.RuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionRequest)
		if matchedRule == nil {
			_, _ = decision.Action.Execute(metadata)
			return
		}
		decision.MatchedRule = matchedRule
		decision.Action = matchedRule.Action()
		contine, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
			return
		}
		if !contine {
			break
		}
	}

	return
}

func (r *RuleRewriter) RewriteResponse(metadata *common.Metadata) (decision *RewriteDecision) {
	decision = &RewriteDecision{
		Action: action.DirectAction,
	}

	var matchedRule common.Rule
	index := -1
	for {
		matchedRule, index = r.RuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionResponse)
		if matchedRule == nil {
			_, _ = decision.Action.Execute(metadata)
			return
		}
		decision.MatchedRule = matchedRule
		decision.Action = matchedRule.Action()
		contine, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
			return
		}
		if !contine {
			break
		}
	}

	return
}

func (r *RuleRewriter) ServeRequest() bool {
	return r.RuleEngine.ServeRequest
}

func (r *RuleRewriter) ServeResponse() bool {
	return r.RuleEngine.ServeResponse
}

func NewRuleRewriter(cfg *config.Config, recorder *statistics.Recorder) (*RuleRewriter, error) {
	ruleEngine, err := rule.NewEngine(cfg.RulesJson, &cfg.Rules, recorder)
	if err != nil {
		return nil, fmt.Errorf("rule.NewEngine: %w", err)
	}
	return &RuleRewriter{
		RuleEngine: ruleEngine,
		Recorder:   recorder,
	}, nil
}
