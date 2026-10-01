package rewrite

import (
	"bytes"
	"net"
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

func TestURLRulesAfterHeaderStage(t *testing.T) {
	cases := []struct {
		name    string
		headers []config.Rule
		added   bool
	}{
		{name: "URL-only"},
		{name: "unmatched-header", headers: []config.Rule{{Type: "HEADER-KEYWORD", MatchHeader: "X-Match", MatchValue: "yes", Action: "DIRECT", RewriteDirection: "REQUEST"}}},
		{name: "response-header-only", headers: []config.Rule{{Type: "FINAL", Action: "REPLACE", RewriteDirection: "RESPONSE", RewriteHeader: "X-Response", RewriteValue: "unused"}}},
		{name: "continued-header-exhausted", headers: []config.Rule{{Type: "FINAL", Action: "ADD", RewriteDirection: "REQUEST", RewriteHeader: "X-Added", RewriteValue: "once", Continue: true}}, added: true},
	}
	for _, tc := range cases {
		for _, match := range []bool{false, true} {
			name := tc.name + "/unmatched-URL"
			path := "/other"
			if match {
				name = tc.name + "/matched-URL"
				path = "/echo"
			}
			t.Run(name, func(t *testing.T) {
				cfg := &config.Config{
					HeaderRules:      tc.headers,
					URLRedirectRules: []config.Rule{{Type: "URL-REGEX", MatchValue: "/echo$", Action: "REDIRECT-HEADER", RewriteRegex: "/echo$", RewriteValue: "/destination"}},
				}
				r, err := NewRuleRewriter(cfg, nil)
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest("GET", "http://example.com"+path, nil)
				decision := r.RewriteRequest(&common.Metadata{Request: req})
				if match {
					if req.URL.Path != "/destination" || decision.MatchedRule == nil || decision.Action.Type() != common.ActionRedirectHeader {
						t.Fatal("URL rule was not executed after the Header stage")
					}
				} else if req.URL.Path != path || decision.Action != action.DirectAction || decision.Redirect {
					t.Fatal("unmatched URL rule changed the request or decision")
				}
				values := req.Header.Values("X-Added")
				if tc.added && (len(values) != 1 || values[0] != "once") {
					t.Fatalf("continued Header action executed more than once: %v", values)
				}
			})
		}
	}
}

type redirectCaptureConn struct {
	net.Conn
	written bytes.Buffer
}

func (c *redirectCaptureConn) Write(p []byte) (int, error) {
	return c.written.Write(p)
}

func (c *redirectCaptureConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.0.2.2"), Port: 80}
}

func TestURLOnlyRedirectResponseActions(t *testing.T) {
	for _, terminal := range []string{"REDIRECT-302", "REDIRECT-307"} {
		t.Run(terminal, func(t *testing.T) {
			cfg := &config.Config{URLRedirectRules: []config.Rule{{Type: "URL-REGEX", MatchValue: "/echo$", Action: terminal, RewriteRegex: "/echo$", RewriteValue: "/destination"}}}
			r, err := NewRuleRewriter(cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			conn := &redirectCaptureConn{}
			metadata := &common.Metadata{
				Request:  httptest.NewRequest("GET", "http://example.com/echo", nil),
				ConnLink: &common.ConnLink{LConn: conn, RConn: conn, LAddr: "192.0.2.1:1234", RAddr: "example.com:80"},
			}
			decision := r.RewriteRequest(metadata)
			response := conn.written.String()
			if decision.MatchedRule == nil || !strings.HasPrefix(response, "HTTP/1.1 "+terminal[len("REDIRECT-"):]) || !strings.Contains(response, "Location: http://example.com/destination\r\n") {
				t.Fatalf("URL-only redirect action was not executed: %q", response)
			}
		})
	}
}
