package rewrite

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type RuleRewriter struct {
	HeaderRuleEngine  *rule.Engine
	BodyRuleEngine    *rule.Engine
	URLRedirectEngine *rule.Engine
	Recorder          *statistics.Recorder
}

func (r *RuleRewriter) RewriteRequest(metadata *common.Metadata) (decision *RewriteDecision) {
	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	var matchedRule common.Rule

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}
	matchedRule = nil
	index := -1
	for {
		matchedRule, index = r.BodyRuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionRequest)
		if matchedRule == nil {
			break
		}
		decision.Action = matchedRule.Action()
		contine, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
		}
		if !contine {
			break
		}
	}

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}
	matchedRule = nil
	index = -1
	for {
		matchedRule, index = r.HeaderRuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionRequest)
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

	decision = &RewriteDecision{
		Action:   action.DirectAction,
		Continue: true,
	}
	matchedRule = nil
	index = -1
	for {
		matchedRule, index = r.URLRedirectEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionRequest)
		if matchedRule == nil {
			return
		}
		decision.MatchedRule = matchedRule
		decision.Action = matchedRule.Action()
		contine, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
			return
		}
		decision.Continue = contine
		if !contine {
			break
		}
	}

	return
}

func (r *RuleRewriter) RewriteResponse(metadata *common.Metadata) (decision *RewriteDecision) {
	var matchedRule common.Rule

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}
	matchedRule = nil
	index := -1
	for {
		matchedRule, index = r.BodyRuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionResponse)
		if matchedRule == nil {
			break
		}
		decision.Action = matchedRule.Action()
		contine, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
		}
		if !contine {
			break
		}
	}

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}
	matchedRule = nil
	index = -1
	for {
		matchedRule, index = r.HeaderRuleEngine.MatchWithRuleIndex(metadata, index+1, common.DirectionResponse)
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
	return r.HeaderRuleEngine.ServeRequest || r.BodyRuleEngine.ServeRequest
}

func (r *RuleRewriter) ServeResponse() bool {
	return r.HeaderRuleEngine.ServeResponse || r.BodyRuleEngine.ServeResponse
}

func NewRuleRewriter(cfg *config.Config, recorder *statistics.Recorder) (*RuleRewriter, error) {
	headerRuleEngine, err := rule.NewEngine(cfg.HeaderRulesJson, &cfg.HeaderRules, recorder, common.ActionTargetHeader)
	if err != nil {
		return nil, fmt.Errorf("rule.NewEngine: %w", err)
	}
	slog.Info("Header Rule Engine initialized", "rules_count", headerRuleEngine.RulesCount(), "serve_request", headerRuleEngine.ServeRequest, "serve_response", headerRuleEngine.ServeResponse)

	bodyRuleEngine, err := rule.NewEngine(cfg.BodyRulesJson, &cfg.BodyRules, recorder, common.ActionTargetBody)
	if err != nil {
		return nil, fmt.Errorf("rule.NewEngine: %w", err)
	}
	slog.Info("Body Rule Engine initialized", "rules_count", bodyRuleEngine.RulesCount(), "serve_request", bodyRuleEngine.ServeRequest, "serve_response", bodyRuleEngine.ServeResponse)

	redirectRuleEngine, err := rule.NewEngine(cfg.URLRedirectJson, &cfg.URLRedirectRules, recorder, common.ActionTargetURL)
	if err != nil {
		return nil, fmt.Errorf("rule.NewEngine: %w", err)
	}
	slog.Info("URL Redirect Rule Engine initialized", "rules_count", redirectRuleEngine.RulesCount(), "serve_request", redirectRuleEngine.ServeRequest, "serve_response", redirectRuleEngine.ServeResponse)

	return &RuleRewriter{
		HeaderRuleEngine:  headerRuleEngine,
		BodyRuleEngine:    bodyRuleEngine,
		URLRedirectEngine: redirectRuleEngine,
		Recorder:          recorder,
	}, nil
}
