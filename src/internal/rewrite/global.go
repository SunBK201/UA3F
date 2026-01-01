package rewrite

import (
	"fmt"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type GlobalRewriter struct {
	UserAgent      string
	uaRegex        *regexp2.Regexp
	partialReplace bool
	rewriteAction  common.Action

	whitelist []string

	Recorder *statistics.Recorder
}

func (r *GlobalRewriter) RewriteRequest(metadata *common.Metadata) (decision *RewriteDecision) {
	defer func() {
		_, err := decision.Action.Execute(metadata)
		if err != nil {
			log.LogErrorWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("decision.Action.Execute: %s", err.Error()))
		}
		log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Rewrite decision: Action=%s, NeedCache=%v, NeedSkip=%v", decision.Action.Type(), decision.NeedCache, decision.NeedSkip))
	}()

	ua := metadata.UserAgent()
	log.LogInfoWithAddr(metadata.SrcAddr(), metadata.DestAddr(), fmt.Sprintf("Original User-Agent: (%s)", ua))

	decision = &RewriteDecision{
		Action: action.DirectAction,
	}

	if ua == "" {
		return
	}

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

func (r *GlobalRewriter) RewriteResponse(metadata *common.Metadata) (decision *RewriteDecision) {
	return &RewriteDecision{
		Action: action.DirectAction,
	}
}

func (r *GlobalRewriter) ServeRequest() bool {
	return true
}

func (r *GlobalRewriter) ServeResponse() bool {
	return false
}

func (r *GlobalRewriter) inWhitelist(ua string) bool {
	for _, w := range r.whitelist {
		if w == ua {
			return true
		}
	}
	return false
}

func NewGlobalRewriter(cfg *config.Config, recorder *statistics.Recorder) (*GlobalRewriter, error) {
	var err error
	var regex *regexp2.Regexp

	if cfg.UserAgentRegex != "" {
		regex, err = regexp2.Compile("(?i)"+cfg.UserAgentRegex, regexp2.None)
		if err != nil {
			return nil, err
		}
	}

	var rewriteAction common.Action
	if cfg.UserAgentPartialReplace && cfg.UserAgentRegex != "" {
		rewriteAction = action.NewReplaceRegex(recorder, "User-Agent", cfg.UserAgentRegex, cfg.UserAgent, false, common.DirectionRequest)
	} else {
		rewriteAction = action.NewReplace(recorder, "User-Agent", cfg.UserAgent, false, common.DirectionRequest)
	}
	if rewriteAction == nil {
		return nil, fmt.Errorf("failed to create rewrite action")
	}

	return &GlobalRewriter{
		UserAgent:      cfg.UserAgent,
		uaRegex:        regex,
		partialReplace: cfg.UserAgentPartialReplace,
		rewriteAction:  rewriteAction,
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
