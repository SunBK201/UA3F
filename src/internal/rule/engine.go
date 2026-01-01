package rule

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/match"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Engine struct {
	rules         []common.Rule
	ServeRequest  bool
	ServeResponse bool
}

func NewEngine(rulesJSON string, ruleSet *[]config.Rule, recorder *statistics.Recorder) (*Engine, error) {
	var (
		rules    []common.Rule
		rulesCfg []*config.Rule
	)

	if ruleSet != nil && len(*ruleSet) > 0 {
		for i := range *ruleSet {
			(*ruleSet)[i].Enabled = true
			rulesCfg = append(rulesCfg, &(*ruleSet)[i])
		}
	} else {
		if rulesJSON == "" {
			return &Engine{rules: []common.Rule{}}, nil
		}
		if err := json.Unmarshal([]byte(rulesJSON), &rulesCfg); err != nil {
			return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
		}
	}

	action.InitActions(recorder)
	validate := validator.New()

	var r common.Rule
	for _, rule := range rulesCfg {
		if !rule.Enabled {
			continue
		}

		if err := validate.Struct(rule); err != nil {
			slog.Warn("Invalid rule", slog.Any("rule", rule), slog.Any("error", err))
			rule.Enabled = false
			continue
		}

		switch common.RuleType(rule.Type) {
		case common.RuleTypeHeaderKeyword:
			r = match.NewHeaderKeyword(rule, recorder)
		case common.RuleTypeHeaderRegex:
			r = match.NewHeaderRegex(rule, recorder)
		case common.RuleTypeIPCIDR:
			r = match.NewIPCIDR(rule, recorder)
		case common.RuleTypeSrcIP:
			r = match.NewSrcIP(rule, recorder)
		case common.RuleTypeDestPort:
			r = match.NewDestPort(rule, recorder)
		case common.RuleTypeDomain:
			r = match.NewDomain(rule, recorder)
		case common.RuleTypeDomainKeyword:
			r = match.NewDomainKeyword(rule, recorder)
		case common.RuleTypeDomainSuffix:
			r = match.NewDomainSuffix(rule, recorder)
		case common.RuleTypeURLRegex:
			r = match.NewURLRegex(rule, recorder)
		case common.RuleTypeFinal:
			r = match.NewFinal(rule, recorder)
		default:
			slog.Warn("Unsupported rule type", slog.String("type", rule.Type))
			continue
		}
		if r != nil {
			rules = append(rules, r)
		}
	}

	var serveRequest, serveResponse bool
	for _, rule := range rules {
		if rule.Action().Direction() == common.DirectionRequest {
			serveRequest = true
		}
		if rule.Action().Direction() == common.DirectionResponse {
			serveResponse = true
		}
	}

	return &Engine{rules: rules, ServeRequest: serveRequest, ServeResponse: serveResponse}, nil
}

func (e *Engine) MatchWithRule(metadata *common.Metadata) common.Rule {
	for _, rule := range e.rules {
		matched := rule.Match(metadata)
		if matched {
			slog.Info("Rule matched", slog.Any("rule", rule), slog.Any("metadata", metadata))
			return rule
		}
	}
	slog.Warn("No rule matched", slog.Any("metadata", metadata))
	return nil
}

func (e *Engine) MatchWithRuleIndex(metadata *common.Metadata, startIndex int, direction common.Direction) (common.Rule, int) {
	if startIndex < 0 || startIndex >= len(e.rules) {
		return nil, -1
	}
	for i := startIndex; i < len(e.rules); i++ {
		rule := e.rules[i]
		if rule.Action().Direction() != common.DirectionDual && rule.Action().Direction() != direction {
			continue
		}
		matched := rule.Match(metadata)
		if matched {
			slog.Info("Rule matched", slog.Any("rule", rule), slog.Any("metadata", metadata))
			return rule, i
		}
	}
	slog.Warn("No rule matched", slog.Any("metadata", metadata))
	return nil, -1
}
