package rewrite

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Rewriter interface {
	Rewrite(metadata *common.Metadata) (decision *RewriteDecision)
}

type RewriteDecision struct {
	Action      common.Action
	MatchedRule common.Rule
	NeedCache   bool
	NeedSkip    bool

	Modified bool // NFQUEUE
	HasUA    bool // NFQUEUE
}

func New(cfg *config.Config, recorder *statistics.Recorder) (Rewriter, error) {
	if cfg.ServerMode == config.ServerModeNFQueue {
		return NewPacketRewriter(cfg, recorder)
	}

	switch cfg.RewriteMode {
	case config.RewriteModeDirect:
		return NewDirectRewriter(), nil
	case config.RewriteModeGlobal:
		return NewGlobalRewriter(cfg, recorder)
	case config.RewriteModeRule:
		return NewRuleRewriter(cfg, recorder)
	default:
		return nil, nil
	}
}
