package rewrite

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dlclark/regexp2"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/rule/common"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	payloadUA      string
	pattern        string
	partialReplace bool
	rewriteMode    config.RewriteMode
	rewriteAction  common.Action

	uaRegex    *regexp2.Regexp
	ruleEngine *rule.Engine
	whitelist  []string

	Recorder *statistics.Recorder
}

type RewriteDecision struct {
	Action      common.Action
	MatchedRule common.Rule
	NeedCache   bool
	NeedSkip    bool
}

func (d *RewriteDecision) ShouldRewrite() bool {
	if d.NeedCache || d.NeedSkip {
		return false
	}
	return d.Action.Type() == common.ActionReplace ||
		d.Action.Type() == common.ActionReplacePart ||
		d.Action.Type() == common.ActionDelete
}

func New(cfg *config.Config, recorder *statistics.Recorder) (*Rewriter, error) {
	pattern := "(?i)" + cfg.UserAgentRegex // case-insensitive prefix (?i)
	uaRegex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	var ruleEngine *rule.Engine
	if cfg.RewriteMode == config.RewriteModeRule {
		ruleEngine, err = rule.NewEngine(cfg.RulesJson, &cfg.Rules)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule engine: %w", err)
		}
	}

	var rewriteAction common.Action
	if cfg.RewriteMode == config.RewriteModeGlobal {
		if cfg.UserAgentPartialReplace && cfg.UserAgentRegex != "" {
			rewriteAction = action.NewReplacePart(cfg.UserAgentRegex, "User-Agent", cfg.UserAgent, true)
		} else {
			rewriteAction = action.NewReplace("User-Agent", cfg.UserAgent)
		}
		if rewriteAction == nil {
			return nil, fmt.Errorf("failed to create rewrite action")
		}
	}

	return &Rewriter{
		payloadUA:      cfg.UserAgent,
		pattern:        cfg.UserAgentRegex,
		partialReplace: cfg.UserAgentPartialReplace,
		rewriteMode:    cfg.RewriteMode,
		rewriteAction:  rewriteAction,
		uaRegex:        uaRegex,
		ruleEngine:     ruleEngine,
		whitelist: []string{
			"MicroMessenger Client",
			"Bilibili Freedoooooom/MarkII",
			"Valve/Steam HTTP Client 1.0",
			"Go-http-client/1.1",
			"ByteDancePcdn",
		},
		Recorder: recorder,
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
			slog.Error("r.uaRegex.Replace", slog.Any("error", err))
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
		r.Recorder.AddRecord(&statistics.PassThroughRecord{
			SrcAddr:  srcAddr,
			DestAddr: destAddr,
			UA:       originalUA,
		})
		return &RewriteDecision{
			Action: action.DirectAction,
		}
	}

	// RULE
	if r.rewriteMode == config.RewriteModeRule {
		matchedRule := r.ruleEngine.MatchWithRule(req, srcAddr, destAddr)

		// no match rule, direct forward
		if matchedRule == nil {
			log.LogDebugWithAddr(srcAddr, destAddr, "No rule matched, direct forward")
			r.Recorder.AddRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
				UA:       originalUA,
			})
			return &RewriteDecision{
				Action: action.DirectAction,
			}
		}

		// DROP
		if matchedRule.Action() == action.DropAction {
			log.LogInfoWithAddr(srcAddr, destAddr, "Rule matched: DROP action, request will be dropped")
			return &RewriteDecision{
				Action:      matchedRule.Action(),
				MatchedRule: matchedRule,
			}
		}

		// DIRECT
		if matchedRule.Action() == action.DirectAction {
			log.LogDebugWithAddr(srcAddr, destAddr, "Rule matched: DIRECT action, skip rewriting")
			r.Recorder.AddRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
				UA:       originalUA,
			})
			return &RewriteDecision{
				Action:      matchedRule.Action(),
				MatchedRule: matchedRule,
			}
		}

		// REPLACE、REPLACE-PART、DELETE, Rewrite
		return &RewriteDecision{
			Action:      matchedRule.Action(),
			MatchedRule: matchedRule,
		}
	}

	// GLOBAL
	var err error
	matches := false
	decision := &RewriteDecision{}

	if originalUA == "" {
		decision.Action = action.DirectAction
		return decision
	}

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
		decision.NeedCache = true
		if originalUA == "Valve/Steam HTTP Client 1.0" {
			decision.NeedSkip = true
		}
	}
	if !matches {
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Not hit User-Agent regex: %s", originalUA))
	}

	hit := !isWhitelist && matches
	if !hit {
		r.Recorder.AddRecord(&statistics.PassThroughRecord{
			SrcAddr:  srcAddr,
			DestAddr: destAddr,
			UA:       originalUA,
		})
		decision.Action = action.DirectAction
		return decision
	}
	decision.Action = r.rewriteAction
	return decision
}

func (r *Rewriter) Rewrite(req *http.Request, srcAddr string, destAddr string, decision *RewriteDecision) *http.Request {
	originalValue, rewritedValue := decision.Action.Execute(&common.Metadata{
		Request:  req,
		SrcAddr:  srcAddr,
		DestAddr: destAddr,
	})

	r.Recorder.AddRecord(&statistics.RewriteRecord{
		Host:       destAddr,
		OriginalUA: originalValue,
		MockedUA:   rewritedValue,
	})
	return req
}
