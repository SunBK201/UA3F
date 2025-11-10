package rule

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"
)

type RuleType string

const (
	RuleTypeKeyword  RuleType = "KEYWORD"
	RuleTypeRegex    RuleType = "REGEX"
	RuleTypeIPCIDR   RuleType = "IP-CIDR"
	RuleTypeSrcIP    RuleType = "SRC-IP"
	RuleTypeDestPort RuleType = "DEST-PORT"
	RuleTypeFinal    RuleType = "FINAL"
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
	Enabled      bool     `json:"enabled"`
	Type         RuleType `json:"type"`
	Action       Action   `json:"action"`
	MatchValue   string   `json:"match_value"`
	RewriteValue string   `json:"rewrite_value"`
	Description  string   `json:"description"`

	regex *regexp2.Regexp
	ipNet *net.IPNet
}

type Engine struct {
	rules []*Rule
}

func NewEngine(rulesJSON string) (*Engine, error) {
	if rulesJSON == "" {
		return &Engine{rules: []*Rule{}}, nil
	}

	var rules []*Rule
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case RuleTypeRegex:
			if rule.MatchValue != "" {
				pattern := "(?i)" + rule.MatchValue
				regex, err := regexp2.Compile(pattern, regexp2.None)
				if err != nil {
					logrus.Warnf("Failed to compile regex for rule: %s, error: %v", rule.Description, err)
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
					logrus.Warnf("Failed to parse CIDR for rule: %s, error: %v", rule.Description, err)
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
		case RuleTypeKeyword:
			matched = e.matchKeyword(req, rule)
		case RuleTypeRegex:
			matched, err = e.matchRegex(req, rule)
			if err != nil {
				logrus.Warnf("Regex match error: %v", err)
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
			logrus.Debugf("Rule matched: %s (type: %s, action: %s)", rule.Description, rule.Type, rule.Action)
			return rule
		}
	}
	return nil
}

func (e *Engine) matchKeyword(req *http.Request, rule *Rule) bool {
	ua := req.Header.Get("User-Agent")
	return strings.Contains(strings.ToLower(ua), strings.ToLower(rule.MatchValue))
}

func (e *Engine) matchRegex(req *http.Request, rule *Rule) (bool, error) {
	if rule.regex == nil {
		return false, nil
	}
	ua := req.Header.Get("User-Agent")
	return rule.regex.MatchString(ua)
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
		if rule != nil && rule.Type == RuleTypeRegex && rule.regex != nil {
			newUA, err := rule.regex.Replace(originalUA, rewriteValue, -1, -1)
			if err != nil {
				logrus.Errorf("Failed to apply REPLACE-PART: %v, using full replacement", err)
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
