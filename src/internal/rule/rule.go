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

// RuleType 规则类型
type RuleType string

const (
	RuleTypeKeyword  RuleType = "KEYWORD"   // 关键字匹配
	RuleTypeRegex    RuleType = "REGEX"     // 正则表达式匹配
	RuleTypeIPCIDR   RuleType = "IP-CIDR"   // IP地址段匹配
	RuleTypeSrcIP    RuleType = "SRC-IP"    // 源IP地址匹配
	RuleTypeDestPort RuleType = "DEST-PORT" // 目标端口匹配
	RuleTypeFinal    RuleType = "FINAL"     // 兜底规则
)

// Action 重写策略
type Action string

const (
	ActionReplace     Action = "REPLACE"      // 替换整个 User-Agent
	ActionReplacePart Action = "REPLACE-PART" // 部分替换
	ActionDelete      Action = "DELETE"       // 删除 User-Agent
	ActionDirect      Action = "DIRECT"       // 直接转发
	ActionDrop        Action = "DROP"         // 丢弃请求
)

// Rule 重写规则
type Rule struct {
	Enabled      bool     `json:"enabled"`       // 是否启用
	Type         RuleType `json:"type"`          // 规则类型
	Action       Action   `json:"action"`        // 重写策略
	MatchValue   string   `json:"match_value"`   // 匹配值
	RewriteValue string   `json:"rewrite_value"` // 重写值
	Description  string   `json:"description"`   // 描述

	// 编译后的正则表达式（仅用于 REGEX 类型）
	regex *regexp2.Regexp
	// 解析后的 IP 网络（仅用于 IP-CIDR 和 SRC-IP 类型）
	ipNet *net.IPNet
}

// Engine 规则引擎
type Engine struct {
	rules []*Rule
}

// NewEngine 创建规则引擎
func NewEngine(rulesJSON string) (*Engine, error) {
	if rulesJSON == "" {
		return &Engine{rules: []*Rule{}}, nil
	}

	var rules []*Rule
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	// 编译正则表达式和解析 IP 网络
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

// MatchWithRule 匹配规则并返回匹配的规则
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

// matchKeyword 关键字匹配
func (e *Engine) matchKeyword(req *http.Request, rule *Rule) bool {
	ua := req.Header.Get("User-Agent")
	return strings.Contains(strings.ToLower(ua), strings.ToLower(rule.MatchValue))
}

// matchRegex 正则表达式匹配
func (e *Engine) matchRegex(req *http.Request, rule *Rule) (bool, error) {
	if rule.regex == nil {
		return false, nil
	}
	ua := req.Header.Get("User-Agent")
	return rule.regex.MatchString(ua)
}

// matchIPCIDR 目标IP地址段匹配
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

// matchSrcIP 源IP地址匹配
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

// matchDestPort 目标端口匹配
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

// ApplyAction 应用规则动作
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
		// 如果不是正则规则，使用简单字符串替换
		return strings.ReplaceAll(originalUA, rule.MatchValue, rewriteValue)
	case ActionDelete:
		return ""
	case ActionDirect:
		return originalUA
	case ActionDrop:
		return originalUA // DROP 由上层处理
	default:
		return originalUA
	}
}

// HasRules 是否有启用的规则
func (e *Engine) HasRules() bool {
	for _, rule := range e.rules {
		if rule.Enabled {
			return true
		}
	}
	return false
}

// GetRules 获取所有规则
func (e *Engine) GetRules() []*Rule {
	return e.rules
}
