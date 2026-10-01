package rule

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
)

func TestRuleEnabled(t *testing.T) {
	for _, source := range []string{"yaml", "json"} {
		for _, state := range []string{"omitted", "true", "false"} {
			t.Run(source+"/"+state, func(t *testing.T) {
				cfg := config.Rule{Type: "FINAL", Action: "REPLACE", RewriteHeader: "User-Agent", RewriteValue: "REWRITTEN"}
				if state != "omitted" {
					enabled := state == "true"
					cfg.Enabled = &enabled
				}
				var ruleSet *[]config.Rule
				rulesJSON := ""
				if source == "yaml" {
					rules := []config.Rule{cfg}
					ruleSet = &rules
				} else {
					data, _ := json.Marshal([]config.Rule{cfg})
					rulesJSON = string(data)
				}
				engine, err := NewEngine(rulesJSON, ruleSet, nil, common.ActionTargetHeader)
				if err != nil {
					t.Fatal(err)
				}
				metadata := &common.Metadata{Request: httptest.NewRequest("GET", "http://example.com/", nil)}
				metadata.Request.Header.Set("User-Agent", "ORIGINAL")
				matched, _ := engine.MatchWithRuleIndex(metadata, 0, common.DirectionRequest)
				if state == "false" {
					if engine.RulesCount() != 0 || matched != nil || engine.ServeRequest {
						t.Fatal("disabled rule participates in matching")
					}
					data, _ := json.Marshal(cfg)
					if !strings.Contains(string(data), `"enabled":false`) {
						t.Fatal("explicit false lost during serialization")
					}
				} else {
					if matched == nil || engine.RulesCount() != 1 || !engine.ServeRequest {
						t.Fatal("enabled/default rule was skipped")
					}
					if _, err := matched.Action().Execute(metadata); err != nil {
						t.Fatal(err)
					}
					if got := metadata.Request.Header.Get("User-Agent"); got != "REWRITTEN" {
						t.Fatalf("UA=%q", got)
					}
				}
			})
		}
	}
}

func TestInvalidRulesStillSkipped(t *testing.T) {
	rules := []config.Rule{
		{Type: "FINAL", Action: "UNKNOWN"},
		{Type: "FINAL", Action: "REPLACE", RewriteHeader: "User-Agent", RewriteValue: "REWRITTEN"},
	}
	engine, err := NewEngine("", &rules, nil, common.ActionTargetHeader)
	if err != nil {
		t.Fatal(err)
	}
	if engine.RulesCount() != 1 {
		t.Fatalf("got %d rules", engine.RulesCount())
	}
}
