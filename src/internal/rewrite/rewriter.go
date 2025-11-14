package rewrite

import (
	"fmt"
	"net/http"

	"github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	payloadUA      string
	pattern        string
	partialReplace bool
	rewriteMode    config.RewriteMode

	uaRegex    *regexp2.Regexp
	ruleEngine *rule.Engine
	whitelist  []string
}

type RewriteDecision struct {
	Action      rule.Action
	MatchedRule *rule.Rule
	NeedCache   bool
}

func (d *RewriteDecision) ShouldRewrite() bool {
	if d.NeedCache {
		return false
	}
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
	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Original User-Agent: (%s)", originalUA))
	if originalUA == "" {
		req.Header.Set("User-Agent", "")
	}

	// DIRECT
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

	// RULES
	if r.rewriteMode == config.RewriteModeRules && r.ruleEngine != nil {
		matchedRule := r.ruleEngine.MatchWithRule(req, srcAddr, destAddr)

		// no match rule, direct forward
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

		// DROP
		if matchedRule.Action == rule.ActionDrop {
			log.LogInfoWithAddr(srcAddr, destAddr, "Rule matched: DROP action, request will be dropped")
			return &RewriteDecision{
				Action:      matchedRule.Action,
				MatchedRule: matchedRule,
			}
		}

		// DIRECT
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

		// REPLACE、REPLACE-PART、DELETE, Rewrite
		return &RewriteDecision{
			Action:      matchedRule.Action,
			MatchedRule: matchedRule,
		}
	}

	// GLOBAL
	var err error
	matches := false
	isWhitelist := r.inWhitelist(originalUA)
	decision := &RewriteDecision{}

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
		decision.NeedCache = true
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
		decision.Action = rule.ActionDirect
		return decision
	}
	decision.Action = rule.ActionReplace
	return decision
}

func (r *Rewriter) Rewrite(req *http.Request, srcAddr string, destAddr string, decision *RewriteDecision) *http.Request {
	headerName := "User-Agent"
	if decision.MatchedRule != nil && decision.MatchedRule.RewriteHeader != "" {
		headerName = decision.MatchedRule.RewriteHeader
	}

	originalValue := req.Header.Get(headerName)
	rewriteValue := ""
	if decision.MatchedRule != nil {
		rewriteValue = decision.MatchedRule.RewriteValue
	}
	action := decision.Action
	var rewritedValue string

	// RULES
	if r.rewriteMode == config.RewriteModeRules && r.ruleEngine != nil {
		rewritedValue = r.ruleEngine.ApplyAction(action, rewriteValue, originalValue, decision.MatchedRule)
	} else {
		// GLOBAL
		rewritedValue = r.buildUserAgent(originalValue)
	}

	req.Header.Set(headerName, rewritedValue)

	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Rewrite %s from (%s) to (%s)", headerName, originalValue, rewritedValue))

	statistics.AddRewriteRecord(&statistics.RewriteRecord{
		Host:       destAddr,
		OriginalUA: originalValue,
		MockedUA:   rewritedValue,
	})
	return req
}
