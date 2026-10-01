package rewrite

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
)

func TestRequestTerminalActions(t *testing.T) {
	for _, stage := range []string{"body", "header"} {
		for _, terminal := range []string{"DROP", "REJECT"} {
			for _, laterRules := range []bool{false, true} {
				name := stage + "/" + terminal + "/empty-later-stages"
				if laterRules {
					name = stage + "/" + terminal + "/matching-later-rules"
				}
				t.Run(name, func(t *testing.T) {
					cfg := &config.Config{}
					stop := config.Rule{Type: "FINAL", Action: terminal, RewriteDirection: "REQUEST", Continue: true}
					header := config.Rule{Type: "FINAL", Action: "REPLACE", RewriteDirection: "REQUEST", RewriteHeader: "X-Later", RewriteValue: "executed"}
					if stage == "body" {
						cfg.BodyRules = []config.Rule{stop}
						if laterRules {
							cfg.HeaderRules = []config.Rule{header}
						}
					} else {
						cfg.HeaderRules = []config.Rule{stop}
						if laterRules {
							cfg.HeaderRules = append(cfg.HeaderRules, header)
						}
					}
					if laterRules {
						cfg.URLRedirectRules = []config.Rule{{Type: "FINAL", Action: "REDIRECT-HEADER", RewriteRegex: "/echo", RewriteValue: "/later"}}
					}
					r, err := NewRuleRewriter(cfg, nil)
					if err != nil {
						t.Fatal(err)
					}
					req := httptest.NewRequest("GET", "http://example.com/echo", nil)
					decision := r.RewriteRequest(&common.Metadata{Request: req})
					var want common.Action = action.DropRequestAction
					if terminal == "REJECT" {
						want = action.RejectRequestAction
					}
					if decision.Action != want {
						t.Fatalf("terminal %s decision lost: got %s", terminal, decision.Action.Type())
					}
					if req.Header.Get("X-Later") != "" || req.URL.Path != "/echo" || decision.Redirect {
						t.Fatal("rules after the terminal action executed")
					}
				})
			}
		}
	}
}

func TestUnmatchedRequestDropAllowsLaterRewrites(t *testing.T) {
	for _, stage := range []string{"body", "header"} {
		t.Run(stage, func(t *testing.T) {
			drop := config.Rule{Type: "HEADER-KEYWORD", MatchHeader: "X-Block", MatchValue: "yes", Action: "DROP", RewriteDirection: "REQUEST"}
			cfg := &config.Config{
				BodyRules:        []config.Rule{{Type: "FINAL", Action: "REPLACE-REGEX", RewriteDirection: "REQUEST", RewriteRegex: "original", RewriteValue: "rewritten"}},
				HeaderRules:      []config.Rule{{Type: "FINAL", Action: "REPLACE", RewriteDirection: "REQUEST", RewriteHeader: "X-Later", RewriteValue: "executed"}},
				URLRedirectRules: []config.Rule{{Type: "FINAL", Action: "REDIRECT-HEADER", RewriteRegex: "/echo", RewriteValue: "/later"}},
			}
			if stage == "body" {
				cfg.BodyRules = append([]config.Rule{drop}, cfg.BodyRules...)
			} else {
				cfg.HeaderRules = append([]config.Rule{drop}, cfg.HeaderRules...)
			}
			r, err := NewRuleRewriter(cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			metadata := &common.Metadata{Request: httptest.NewRequest("POST", "http://example.com/echo", strings.NewReader("original"))}
			decision := r.RewriteRequest(metadata)
			if decision.Action == action.DropRequestAction {
				t.Fatal("unmatched DROP blocked the request")
			}
			if metadata.Request.Header.Get("X-Later") != "executed" || metadata.Request.URL.Path != "/later" || string(metadata.RequestBody(false)) != "rewritten" {
				t.Fatal("unmatched DROP interfered with later rewrites")
			}
		})
	}
}
