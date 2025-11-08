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

var one = make([]byte, 1)

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
	pattern := "(?i)" + cfg.UARegex
	uaRegex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	return &Rewriter{
		payloadUA:      cfg.PayloadUA,
		pattern:        cfg.UARegex,
		partialReplace: cfg.PartialReplace,
		uaRegex:        uaRegex,
		Cache:          expirable.NewLRU[string, struct{}](1024, nil, 30*time.Minute),
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
	log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("original User-Agent: (%s)", originalUA))
	if originalUA == "" {
		req.Header.Set("User-Agent", "")
	}

	var err error
	matches := false
	isWhitelist := r.inWhitelist(originalUA)

	if !isWhitelist {
		if r.pattern == "" {
			matches = true
		} else {
			matches, err = r.uaRegex.MatchString(originalUA)
			if err != nil {
				log.LogErrorWithAddr(srcAddr, destAddr, fmt.Sprintf("User-Agent regex match error: %s", err.Error()))
				matches = true
			}
		}
	}

	if isWhitelist {
		log.LogInfoWithAddr(srcAddr, destAddr, fmt.Sprintf("hit User-Agent whitelist: %s, add to cache", originalUA))
		r.Cache.Add(destAddr, struct{}{})
	}
	if !matches {
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("not hit User-Agent regex: %s", originalUA))
	}

	hit := !isWhitelist && matches
	if !hit {
		statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
			SrcAddr:  srcAddr,
			DestAddr: destAddr,
			UA:       originalUA,
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
	reader := bufio.NewReaderSize(src, 64*1024)

	defer func() {
		if err != nil {
			log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Process: %s", err.Error()))
		}
		if _, err = io.CopyBuffer(dst, reader, one); err != nil {
			log.LogWarnWithAddr(srcAddr, destAddr, fmt.Sprintf("Process io.Copy: %s", err.Error()))
		}
	}()

	if strings.HasSuffix(destAddr, "443") {
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			r.Cache.Add(destAddr, struct{}{})
			log.LogInfoWithAddr(srcAddr, destAddr, "tls client hello detected, added to cache")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.HTTPS,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
			return
		}
	}

	var isHTTP bool

	if isHTTP, err = sniff.SniffHTTP(reader); err != nil {
		err = fmt.Errorf("sniff.SniffHTTP: %w", err)
		return
	}
	if !isHTTP {
		r.Cache.Add(destAddr, struct{}{})
		log.LogInfoWithAddr(srcAddr, destAddr, "sniff first request is not http, added to cache, switching to raw proxy")
		if isTLS, _ := sniff.SniffTLS(reader); isTLS {
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.TLS,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
		}
		return
	}

	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol: sniff.HTTP,
		SrcAddr:  srcAddr,
		DestAddr: destAddr,
	})

	var req *http.Request

	for {
		if isHTTP, err = sniff.SniffHTTPFast(reader); err != nil {
			err = fmt.Errorf("sniff.SniffHTTPFast: %w", err)
			statistics.AddConnection(
				&statistics.ConnectionRecord{
					Protocol: sniff.TCP,
					SrcAddr:  srcAddr,
					DestAddr: destAddr,
				},
			)
			return
		}
		if !isHTTP {
			log.LogWarnWithAddr(srcAddr, destAddr, "sniff subsequent request is not http, switching to raw proxy")
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
			log.LogInfoWithAddr(srcAddr, destAddr, "websocket upgrade detected, switching to raw proxy")
			statistics.AddConnection(&statistics.ConnectionRecord{
				Protocol: sniff.WebSocket,
				SrcAddr:  srcAddr,
				DestAddr: destAddr,
			})
			return
		}
	}
}
