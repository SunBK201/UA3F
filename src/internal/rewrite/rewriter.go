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
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	payloadUA      string
	pattern        string
	partialReplace bool

	uaRegex   *regexp2.Regexp
	whitelist []string
	Cache     *expirable.LRU[string, struct{}]
}

// New constructs a Rewriter from config. Compiles regex and allocates cache.
func New(cfg *config.Config) (*Rewriter, error) {
	// UA pattern is compiled with case-insensitive prefix (?i)
	pattern := "(?i)" + cfg.UAPattern
	uaRegex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	return &Rewriter{
		payloadUA:      cfg.PayloadUA,
		pattern:        cfg.UAPattern,
		partialReplace: cfg.EnablePartialReplace,
		uaRegex:        uaRegex,
		Cache:          expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
		whitelist: []string{
			"MicroMessenger Client",
			"Bilibili Freedoooooom/MarkII",
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

// buildUserAgent returns either a partial replacement (regex) or full overwrite.
func (r *Rewriter) buildUserAgent(originUA string) string {
	if r.partialReplace && r.uaRegex != nil && r.pattern != "" {
		newUA, err := r.uaRegex.Replace(originUA, r.payloadUA, -1, -1)
		if err != nil {
			logrus.Errorf("User-Agent Replace Error: %s, use full overwrite", err.Error())
			return r.payloadUA
		}
		return newUA
	}
	return r.payloadUA
}

func (r *Rewriter) ShouldRewrite(req *http.Request, srcAddr, destAddr string) bool {
	originalUA := req.Header.Get("User-Agent")
	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Original User-Agent: (%s)", originalUA))

	var err error
	matches := true
	isWhitelist := r.inWhitelist(originalUA)

	if !isWhitelist && r.pattern != "" {
		matches, err = r.uaRegex.MatchString(originalUA)
		if err != nil {
			log.LogErrorWithAddr(srcAddr, destAddr, fmt.Sprintf("User-Agent Regex Match Error: %s", err.Error()))
			matches = true
		}
	}
	if isWhitelist {
		log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("Hit User-Agent Whitelist: %s", originalUA))
		r.Cache.Add(destAddr, struct{}{})
	}
	if !matches {
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Not Hit User-Agent Regex: %s", originalUA))
	}

	hit := !isWhitelist && matches
	if !hit {
		statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
			Host: destAddr,
			UA:   originalUA,
		})
	}
	return hit
}

func (r *Rewriter) Rewrite(req *http.Request, srcAddr string, destAddr string) *http.Request {
	originalUA := req.Header.Get("User-Agent")
	rewritedUA := r.buildUserAgent(originalUA)
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
	reader := bufio.NewReader(src)

	defer func() {
		if err != nil {
			log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Process: %s", err.Error()))
		}
		io.Copy(dst, reader)
	}()

	if strings.HasSuffix(destAddr, "443") && sniff.SniffTLSClientHello(reader) {
		r.Cache.Add(destAddr, struct{}{})
		log.LogInfoWithAddr(srcAddr, destAddr, "tls client hello detected, pass forward")
		return
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTP(reader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		r.Cache.Add(destAddr, struct{}{})
		log.LogInfoWithAddr(srcAddr, destAddr, "Not HTTP, added to cache")
		return
	}

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("isHTTP: %w", err)
			return
		}
		if !isHTTP {
			r.Cache.Add(destAddr, struct{}{})
			log.LogInfoWithAddr(srcAddr, destAddr, "Not HTTP, added to LRU Relay Cache")
			return
		}
		if req, err = http.ReadRequest(reader); err != nil {
			err = fmt.Errorf("http.ReadRequest: %w", err)
			return
		}
		if r.ShouldRewrite(req, srcAddr, destAddr) {
			req = r.Rewrite(req, srcAddr, destAddr)
		}
		if err = r.Forward(dst, req); err != nil {
			err = fmt.Errorf("r.forward: %w", err)
			return
		}
		if req.Header.Get("Upgrade") == "websocket" && req.Header.Get("Connection") == "Upgrade" {
			log.LogInfoWithAddr(srcAddr, destAddr, "WebSocket Upgrade detected, switching to raw proxy")
			return
		}
	}
}
