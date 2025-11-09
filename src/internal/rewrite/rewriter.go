package rewrite

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

var one = make([]byte, 1)

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	payloadUA      string
	pattern        string
	partialReplace bool
	rewriteMode    config.RewriteMode

	uaRegex    *regexp2.Regexp
	ruleEngine *rule.Engine
	whitelist  []string
	Cache      *expirable.LRU[string, struct{}]
}

// RewriteDecision 重写决策结果
type RewriteDecision struct {
	Action      rule.Action
	MatchedRule *rule.Rule
}

// ShouldRewrite 判断是否需要重写
func (d *RewriteDecision) ShouldRewrite() bool {
	return d.Action == rule.ActionReplace ||
		d.Action == rule.ActionReplacePart ||
		d.Action == rule.ActionDelete
}

// New constructs a Rewriter from config. Compiles regex and allocates cache.
func New(cfg *config.Config) (*Rewriter, error) {
	// UA pattern is compiled with case-insensitive prefix (?i)
	pattern := "(?i)" + cfg.UARegex
	uaRegex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	// 创建规则引擎
	var ruleEngine *rule.Engine
	if cfg.RewriteMode == config.RewriteModeRules {
		ruleEngine, err = rule.NewEngine(cfg.Rules)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule engine: %w", err)
		}
		logrus.Info("Rule engine initialized")
	}

	return &Rewriter{
		payloadUA:      cfg.PayloadUA,
		pattern:        cfg.UARegex,
		partialReplace: cfg.PartialReplace,
		rewriteMode:    cfg.RewriteMode,
		uaRegex:        uaRegex,
		ruleEngine:     ruleEngine,
		Cache:          expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		whitelist: []string{
			"MicroMessenger Client",
			"Bilibili Freedoooooom/MarkII",
			"Valve/Steam HTTP Client 1.0",
			"Go-http-client/1.1",
			"ByteDancePcdn",
		},
	}, nil
}

func (r *Rewriter) inWhitelist(ua string) bool {
	for _, w := range r.whitelist {
		if w == ua {
			return true
		}
	}
	return false
}

// GetRuleEngine 获取规则引擎
func (r *Rewriter) GetRuleEngine() *rule.Engine {
	return r.ruleEngine
}

// buildUserAgent returns either a partial replacement (regex) or full overwrite.
func (r *Rewriter) buildUserAgent(originUA string) string {
	if r.partialReplace && r.uaRegex != nil && r.pattern != "" {
		newUA, err := r.uaRegex.Replace(originUA, r.payloadUA, -1, -1)
		if err != nil {
			logrus.Errorf("r.uaRegex.Replace: %s, use full overwrite", err.Error())
			return r.payloadUA
		}
		return newUA
	}
	return r.payloadUA
}

func (r *Rewriter) EvaluateRewriteDecision(req *http.Request, srcAddr, destAddr string) *RewriteDecision {
	originalUA := req.Header.Get("User-Agent")
	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("original User-Agent: (%s)", originalUA))
	if originalUA == "" {
		req.Header.Set("User-Agent", "")
	}

	// 「直接转发」模式：不进行任何重写
	if r.rewriteMode == config.RewriteModeDirect {
		log.LogDebugWithAddr(srcAddr, destAddr, "Direct forward mode, skip rewriting")
		statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
			SrcAddr:  srcAddr,
			DestAddr: destAddr,
			UA:       originalUA,
		})
		return &RewriteDecision{
			Action: rule.ActionDirect,
		}
	}

	// 「规则判定」模式：使用规则引擎（只匹配一次）
	if r.rewriteMode == config.RewriteModeRules && r.ruleEngine != nil {
		matchedRule := r.ruleEngine.MatchWithRule(req, srcAddr, destAddr)

		// 没有匹配到任何规则，默认直接转发
		if matchedRule == nil {
			log.LogDebugWithAddr(srcAddr, destAddr, "No rule matched, direct forward")
			statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
				UA:       originalUA,
			})
			return &RewriteDecision{
				Action: rule.ActionDirect,
			}
		}

		// DROP 动作：丢弃请求
		if matchedRule.Action == rule.ActionDrop {
			log.LogInfoWithAddr(srcAddr, destAddr, "Rule matched: DROP action, request will be dropped")
			return &RewriteDecision{
				Action:      matchedRule.Action,
				MatchedRule: matchedRule,
			}
		}

		// DIRECT 动作：直接转发
		if matchedRule.Action == rule.ActionDirect {
			log.LogDebugWithAddr(srcAddr, destAddr, "Rule matched: DIRECT action, skip rewriting")
			statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
				UA:       originalUA,
			})
			return &RewriteDecision{
				Action:      matchedRule.Action,
				MatchedRule: matchedRule,
			}
		}

		// REPLACE、REPLACE-PART、DELETE 动作：需要重写
		return &RewriteDecision{
			Action:      matchedRule.Action,
			MatchedRule: matchedRule,
		}
	}

	// 「全局重写」模式：使用原有逻辑
	var err error
	matches := false
	isWhitelist := r.inWhitelist(originalUA)

	if !isWhitelist {
		if r.pattern == "" {
			matches = true
		} else {
			matches, err = r.uaRegex.MatchString(originalUA)
			if err != nil {
				log.LogErrorWithAddr(srcAddr, destAddr, fmt.Sprintf("r.uaRegex.MatchString: %s", err.Error()))
				matches = true
			}
		}
	}

	if isWhitelist {
		log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Hit User-Agent whitelist: %s, add to cache", originalUA))
		r.Cache.Add(destAddr, struct{}{})
	}
	if !matches {
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Not hit User-Agent regex: %s", originalUA))
	}

	hit := !isWhitelist && matches
	if !hit {
		statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
			SrcAddr:  srcAddr,
			DestAddr: destAddr,
			UA:       originalUA,
		})
		return &RewriteDecision{
			Action: rule.ActionDirect,
		}
	}
	return &RewriteDecision{
		Action: rule.ActionReplace,
	}
}

func (r *Rewriter) Rewrite(req *http.Request, srcAddr string, destAddr string, decision *RewriteDecision) *http.Request {
	originalUA := req.Header.Get("User-Agent")
	rewriteValue := decision.MatchedRule.RewriteValue
	action := decision.Action
	var rewritedUA string

	// 「规则判定」模式：根据规则动作决定如何重写
	if r.rewriteMode == config.RewriteModeRules && r.ruleEngine != nil {
		rewritedUA = r.ruleEngine.ApplyAction(action, rewriteValue, originalUA, decision.MatchedRule)
	} else {
		// 「全局重写」模式：使用原有逻辑
		rewritedUA = r.buildUserAgent(originalUA)
	}

	req.Header.Set("User-Agent", rewritedUA)

	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Rewrite User-Agent from (%s) to (%s)", originalUA, rewritedUA))

	statistics.AddRewriteRecord(&statistics.RewriteRecord{
		Host:       destAddr,
		OriginalUA: originalUA,
		MockedUA:   rewritedUA,
	})
	return req
}

func (r *Rewriter) Forward(dst net.Conn, req *http.Request) error {
	if err := req.Write(dst); err != nil {
		return fmt.Errorf("req.Write: %w", err)
	}
	req.Body.Close()
	return nil
}

// Process handles the proxying with UA rewriting logic.
func (r *Rewriter) Process(dst net.Conn, src net.Conn, destAddr string, srcAddr string) (err error) {
	reader := bufio.NewReaderSize(src, 64*1024)

	defer func() {
		if err != nil {
			log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Process: %s", err.Error()))
		}
		if _, err = io.CopyBuffer(dst, reader, one); err != nil {
			log.LogWarnWithAddr(srcAddr, destAddr, fmt.Sprintf("Process io.CopyBuffer: %s", err.Error()))
		}
	}()

	if strings.HasSuffix(destAddr, "443") {
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			r.Cache.Add(destAddr, struct{}{})
			log.LogInfoWithAddr(srcAddr, destAddr, "tls client hello detected, added to cache")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.HTTPS,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
			return
		}
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTP(reader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		r.Cache.Add(destAddr, struct{}{})
		log.LogInfoWithAddr(srcAddr, destAddr, "sniff first request is not http, added to cache, switch to direct forward")
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.TLS,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
		}
		return
	}

	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol: sniff.HTTP,
		SrcAddr:  srcAddr,
		DestAddr: destAddr,
	})

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			statistics.AddConnection(
				&statistics.ConnectionRecord{
					Protocol: sniff.TCP,
					SrcAddr:  srcAddr,
					DestAddr: destAddr,
				},
			)
			return
		}
		if !isHTTP {
			log.LogWarnWithAddr(srcAddr, destAddr, "sniff subsequent request is not http, switch to direct forward")
			return
		}
		if req, err = http.ReadRequest(reader); err != nil {
			err = fmt.Errorf("http.ReadRequest: %w", err)
			return
		}

		// 获取重写决策（只匹配一次规则）
		decision := r.EvaluateRewriteDecision(req, srcAddr, destAddr)
		// 处理 DROP 动作
		if decision.Action == rule.ActionDrop {
			log.LogInfoWithAddr(srcAddr, destAddr, "Request dropped by rule")
			continue
		}
		// 如果需要重写，执行重写操作
		if decision.ShouldRewrite() {
			req = r.Rewrite(req, srcAddr, destAddr, decision)
		}

		if err = r.Forward(dst, req); err != nil {
			err = fmt.Errorf("r.Forward: %w", err)
			return
		}
		if req.Header.Get("Upgrade") == "websocket" && req.Header.Get("Connection") == "Upgrade" {
			log.LogInfoWithAddr(srcAddr, destAddr, "websocket upgrade detected, switch to direct proxy")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.WebSocket,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
			return
		}
	}
}
