package rewrite

import (
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/statistics"
)

func New(cfg *config.Config, recorder *statistics.Recorder) (common.Rewriter, error) {
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
