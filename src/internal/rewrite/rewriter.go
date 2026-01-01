package rewrite

import (
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	UserAgent      string
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
		d.Action.Type() == common.ActionReplaceRegex ||
		d.Action.Type() == common.ActionDelete
}

func New(cfg *config.Config, recorder *statistics.Recorder) (*Rewriter, error) {
	var err error
	var regex *regexp2.Regexp

	if cfg.UserAgentRegex != "" {
		regex, err = regexp2.Compile("(?i)"+cfg.UserAgentRegex, regexp2.None)
		if err != nil {
			return nil, err
		}
	}

	var ruleEngine *rule.Engine
	if cfg.RewriteMode == config.RewriteModeRule {
		ruleEngine, err = rule.NewEngine(cfg.RulesJson, &cfg.Rules, recorder)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule engine: %w", err)
		}
	}

	var rewriteAction common.Action
	if cfg.RewriteMode == config.RewriteModeGlobal {
		if cfg.UserAgentPartialReplace && cfg.UserAgentRegex != "" {
			rewriteAction = action.NewReplaceRegex(recorder, "User-Agent", cfg.UserAgentRegex, cfg.UserAgent)
		} else {
			rewriteAction = action.NewReplace(recorder, "User-Agent", cfg.UserAgent)
		}
		if rewriteAction == nil {
			return nil, fmt.Errorf("failed to create rewrite action")
		}
	}

	return &Rewriter{
		UserAgent:      cfg.UserAgent,
		uaRegex:        regex,
		partialReplace: cfg.UserAgentPartialReplace,
		rewriteMode:    cfg.RewriteMode,
		rewriteAction:  rewriteAction,
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

// buildUserAgent returns either a partial replacement (regex) or full overwrite.
func (r *Rewriter) buildUserAgent(originUA string) string {
	if r.partialReplace && r.uaRegex != nil {
		newUA, err := r.uaRegex.Replace(originUA, r.UserAgent, -1, -1)
		if err != nil {
			slog.Error("r.uaRegex.Replace", slog.Any("error", err))
			return r.UserAgent
		}
		return newUA
	}
	return r.UserAgent
}

func (r *Rewriter) EvaluateRewriteDecision(metadata *common.Metadata) (decision *RewriteDecision) {
	defer func() {
		if decision != nil {
			log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite decision: Action=%s, NeedCache=%v, NeedSkip=%v", decision.Action.Type(), decision.NeedCache, decision.NeedSkip))
		}
	}()

	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	// DIRECT
	if r.rewriteMode == config.RewriteModeDirect {
		return &RewriteDecision{
			Action: action.DirectAction,
		}
	}

	// RULE
	if r.rewriteMode == config.RewriteModeRule {
		matchedRule := r.ruleEngine.MatchWithRule(metadata)
		if matchedRule == nil {
			return &RewriteDecision{
				Action: action.DirectAction,
			}
		}
		return &RewriteDecision{
			Action:      matchedRule.Action(),
			MatchedRule: matchedRule,
		}
	}

	// GLOBAL
	if ua == "" {
		return &RewriteDecision{
			Action: action.DirectAction,
		}
	}

	decision = &RewriteDecision{}

	isWhitelist := r.inWhitelist(ua)
	if isWhitelist {
		decision.Action = action.DirectAction
		decision.NeedCache = true
		if ua == "Valve/Steam HTTP Client 1.0" {
			decision.NeedSkip = true
		}
		log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Hit User-Agent whitelist: %s, add to cache", ua))
		return decision
	}

	if r.uaRegex == nil {
		decision.Action = r.rewriteAction
		return decision
	}

	match, err := r.uaRegex.MatchString(ua)
	if err != nil {
		log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("r.uaRegex.MatchString: %s", err.Error()))
		match = true
	}

	if !match {
		decision.Action = action.DirectAction
		log.LogDebugWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Not hit User-Agent regex: %s", ua))
		return decision
	}

	decision.Action = r.rewriteAction
	return decision
}
