package rule

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/sunbk201/ua3f/internal/config"

	"github.com/dlclark/regexp2"
)

type RuleType string

const (
	RuleTypeHeaderKeyword RuleType = "HEADER-KEYWORD"
	RuleTypeHeaderRegex   RuleType = "HEADER-REGEX"
	RuleTypeIPCIDR        RuleType = "IP-CIDR"
	RuleTypeSrcIP         RuleType = "SRC-IP"
	RuleTypeDestPort      RuleType = "DEST-PORT"
	RuleTypeFinal         RuleType = "FINAL"
)

type Action string

const (
	ActionReplace     Action = "REPLACE"
	ActionReplacePart Action = "REPLACE-PART"
	ActionDelete      Action = "DELETE"
	ActionDirect      Action = "DIRECT"
	ActionDrop        Action = "DROP"
)

type Rule struct {
	Enabled bool `json:"enabled"`

	Type RuleType `json:"type" yaml:"type" validate:"required,oneof=HEADER-KEYWORD HEADER-REGEX DEST-PORT IP-CIDR SRC-IP FINAL"`

	MatchHeader string `json:"match_header,omitempty" yaml:"match-header,omitempty" validate:"required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX"`
	MatchValue  string `json:"match_value,omitempty" yaml:"match-value,omitempty" validate:"required_if=Type DEST-PORT,required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX,required_if=Type IP-CIDR,required_if=Type SRC-IP"`

	Action Action `json:"action" yaml:"action" validate:"required,oneof=DIRECT REPLACE REPLACE-PART DELETE DROP"`

	RewriteHeader string `json:"rewrite_header,omitempty" yaml:"rewrite-header,omitempty" validate:"required_if=Action REPLACE,required_if=Action REPLACE-PART,required_if=Action DELETE"`
	RewriteValue  string `json:"rewrite_value,omitempty" yaml:"rewrite-value,omitempty" validate:"required_if=Action REPLACE,required_if=Action REPLACE-PART"`

	regex *regexp2.Regexp
	ipNet *net.IPNet
}

type Engine struct {
	rules []*Rule
}

func NewEngine(rulesJSON string, ruleSet *[]config.Rule) (*Engine, error) {
	var rules []*Rule

	if ruleSet != nil && len(*ruleSet) > 0 {
		for _, r := range *ruleSet {
			rule := &Rule{
				Enabled:       true,
				Type:          RuleType(r.Type),
				MatchHeader:   r.MatchHeader,
				MatchValue:    r.MatchValue,
				Action:        Action(r.Action),
				RewriteHeader: r.RewriteHeader,
				RewriteValue:  r.RewriteValue,
			}
			rules = append(rules, rule)
		}
	} else {
		if rulesJSON == "" {
			return &Engine{rules: []*Rule{}}, nil
		}
		if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
			return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
		}
	}

	validate := validator.New()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if err := validate.Struct(rule); err != nil {
			slog.Warn("Invalid rule", slog.Any("rule", rule), slog.Any("error", err))
			rule.Enabled = false
			continue
		}

		// Set default RewriteHeader if not specified
		if rule.RewriteHeader == "" {
			rule.RewriteHeader = "User-Agent"
		}

		switch rule.Type {
		case RuleTypeHeaderRegex:
			if rule.MatchValue != "" {
				pattern := "(?i)" + rule.MatchValue
				regex, err := regexp2.Compile(pattern, regexp2.None)
				if err != nil {
					slog.Warn("regexp2.Compile", slog.String("regex", pattern), slog.Any("error", err))
					rule.Enabled = false
					continue
				}
				rule.regex = regex
			}
		case RuleTypeIPCIDR, RuleTypeSrcIP:
			if rule.MatchValue != "" {
				if !strings.Contains(rule.MatchValue, "/") {
					rule.MatchValue += "/32"
				}
				_, ipNet, err := net.ParseCIDR(rule.MatchValue)
				if err != nil {
					slog.Warn("net.ParseCIDR", slog.String("cidr", rule.MatchValue), slog.Any("error", err))
					rule.Enabled = false
					continue
				}
				rule.ipNet = ipNet
			}
		}
	}

	return &Engine{rules: rules}, nil
}

func (e *Engine) MatchWithRule(req *http.Request, srcAddr, destAddr string) *Rule {
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		matched := false
		var err error

		switch rule.Type {
		case RuleTypeHeaderKeyword:
			matched = e.matchHeaderKeyword(req, rule)
		case RuleTypeHeaderRegex:
			matched, err = e.matchHeaderRegex(req, rule)
			if err != nil {
				slog.Warn("e.matchRegex", slog.Any("error", err))
			}
		case RuleTypeIPCIDR:
			matched = e.matchIPCIDR(destAddr, rule)
		case RuleTypeSrcIP:
			matched = e.matchSrcIP(srcAddr, rule)
		case RuleTypeDestPort:
			matched = e.matchDestPort(destAddr, rule)
		case RuleTypeFinal:
			matched = true
		}

		if matched {
			slog.Debug("Rule matched", slog.String("type", string(rule.Type)), slog.String("action", string(rule.Action)))
			return rule
		}
	}
	return nil
}

func (e *Engine) matchHeaderKeyword(req *http.Request, rule *Rule) bool {
	header := req.Header.Get(rule.MatchHeader)
	return strings.Contains(strings.ToLower(header), strings.ToLower(rule.MatchValue))
}

func (e *Engine) matchHeaderRegex(req *http.Request, rule *Rule) (bool, error) {
	if rule.regex == nil {
		return false, nil
	}
	header := req.Header.Get(rule.MatchHeader)
	return rule.regex.MatchString(header)
}

func (e *Engine) matchIPCIDR(destAddr string, rule *Rule) bool {
	if rule.ipNet == nil {
		return false
	}
	host, _, err := net.SplitHostPort(destAddr)
	if err != nil {
		host = destAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return rule.ipNet.Contains(ip)
}

func (e *Engine) matchSrcIP(srcAddr string, rule *Rule) bool {
	if rule.ipNet == nil {
		return false
	}
	host, _, err := net.SplitHostPort(srcAddr)
	if err != nil {
		host = srcAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return rule.ipNet.Contains(ip)
}

func (e *Engine) matchDestPort(destAddr string, rule *Rule) bool {
	_, portStr, err := net.SplitHostPort(destAddr)
	if err != nil {
		return false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}
	matchPort, err := strconv.Atoi(rule.MatchValue)
	if err != nil {
		return false
	}
	return port == matchPort
}

func (e *Engine) ApplyAction(action Action, rewriteValue string, originalUA string, rule *Rule) string {
	switch action {
	case ActionReplace:
		return rewriteValue
	case ActionReplacePart:
		if rule != nil && rule.Type == RuleTypeHeaderRegex && rule.regex != nil {
			newUA, err := rule.regex.Replace(originalUA, rewriteValue, -1, -1)
			if err != nil {
				slog.Error("rule.regex.Replace", slog.Any("error", err))
				return rewriteValue
			}
			return newUA
		}
		// if not regex, do simple string replacement
		return strings.ReplaceAll(originalUA, rule.MatchValue, rewriteValue)
	case ActionDelete:
		return ""
	case ActionDirect:
		return originalUA
	case ActionDrop:
		return originalUA // DROP action handled elsewhere
	default:
		return originalUA
	}
}

func (e *Engine) HasRules() bool {
	for _, rule := range e.rules {
		if rule.Enabled {
			return true
		}
	}
	return false
}

func (e *Engine) GetRules() []*Rule {
	return e.rules
}
