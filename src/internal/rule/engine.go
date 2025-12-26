package rule

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/common"
	"github.com/sunbk201/ua3f/internal/rule/match"
)

type Engine struct {
	rules []common.Rule
}

func NewEngine(rulesJSON string, ruleSet *[]config.Rule) (*Engine, error) {
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
			r = match.NewHeaderKeyword(rule)
		case common.RuleTypeHeaderRegex:
			r = match.NewHeaderRegex(rule)
		case common.RuleTypeIPCIDR:
			r = match.NewIPCIDR(rule)
		case common.RuleTypeSrcIP:
			r = match.NewSrcIP(rule)
		case common.RuleTypeDestPort:
			r = match.NewDestPort(rule)
		case common.RuleTypeDomain:
			r = match.NewDomain(rule)
		case common.RuleTypeDomainKeyword:
			r = match.NewDomainKeyword(rule)
		case common.RuleTypeDomainSuffix:
			r = match.NewDomainSuffix(rule)
		case common.RuleTypeFinal:
			r = match.NewFinal(rule)
		default:
			slog.Warn("Unsupported rule type", slog.String("type", rule.Type))
			continue
		}
		if r != nil {
			rules = append(rules, r)
		}
	}

	return &Engine{rules: rules}, nil
}

func (e *Engine) MatchWithRule(req *http.Request, srcAddr, destAddr string) common.Rule {
	metadata := &common.Metadata{
		Request:  req,
		SrcAddr:  srcAddr,
		DestAddr: destAddr,
	}
	for _, rule := range e.rules {
		matched := rule.Match(metadata)
		if matched {
			slog.Debug("Rule matched", slog.String("type", string(rule.Type())), slog.String("action", string(rule.Action().Type())))
			return rule
		}
	}
	return nil
}
