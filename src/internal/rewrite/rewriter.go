package rewrite

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/statistics"
)

const (
	ErrUseClosedConn   = "use of closed network connection"
	ErrConnResetByPeer = "connection reset by peer"
	ErrIOTimeout       = "i/o timeout"
)

// HTTP methods used to detect HTTP by request line.
var httpMethods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE", "CONNECT"}

// Hardcoded whitelist of UAs that should be left untouched.
var defaultWhitelist = []string{
	"MicroMessenger Client",
	"ByteDancePcdn",
	"Go-http-client/1.1",
}

// Rewriter encapsulates HTTP UA rewrite behavior and pass-through cache.
type Rewriter struct {
	payloadUA            string
	pattern              string
	enablePartialReplace bool

	uaRegex   *regexp2.Regexp
	cache     *expirable.LRU[string, string]
	whitelist map[string]struct{}
}

// New constructs a Rewriter from config. Compiles regex and allocates cache.
func New(cfg *config.Config) (*Rewriter, error) {
	// UA pattern is compiled with case-insensitive prefix (?i)
	pattern := "(?i)" + cfg.UAPattern
	uaRegex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	cache := expirable.NewLRU[string, string](300, nil, 10*time.Minute)

	whitelist := make(map[string]struct{}, len(defaultWhitelist))
	for _, s := range defaultWhitelist {
		whitelist[s] = struct{}{}
	}

	return &Rewriter{
		payloadUA:            cfg.PayloadUA,
		pattern:              cfg.UAPattern,
		enablePartialReplace: cfg.EnablePartialReplace,
		uaRegex:              uaRegex,
		cache:                cache,
		whitelist:            whitelist,
	}, nil
}

// ProxyHTTPOrRaw reads traffic from src and writes to dst.
// - If target in LRU cache: pass-through (raw).
// - Else if HTTP: rewrite UA (unless whitelisted or pattern not matched).
// - Else: mark target in LRU and pass-through.
func (r *Rewriter) ProxyHTTPOrRaw(dst net.Conn, src net.Conn, destAddrPort string) error {
	// Fast path: known pass-through
	if r.cache.Contains(destAddrPort) {
		logrus.Debugf("Hit LRU Relay Cache: %s", destAddrPort)
		_, _ = io.Copy(dst, src)
		return nil
	}

	reader := bufio.NewReader(src)

	isHTTP, err := r.isHTTP(reader)
	if err != nil {
		if strings.Contains(err.Error(), ErrUseClosedConn) {
			logrus.Warnf("[%s] isHTTP error: %s", destAddrPort, err.Error())
			return err
		}
		// Other read errors terminate the direction.
		return err
	}

	if !isHTTP {
		r.cache.Add(destAddrPort, destAddrPort)
		logrus.Debugf("Not HTTP, Add LRU Relay Cache: %s, Cache Len: %d", destAddrPort, r.cache.Len())
		_, _ = io.Copy(dst, reader)
		return nil
	}

	srcAddr := src.RemoteAddr().String()

	// HTTP request loop (handles keep-alive)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			r.logReadErr(destAddrPort, src, err)
			return err
		}

		originalUA := req.Header.Get("User-Agent")

		// No UA header: pass-through after writing this first request
		if originalUA == "" {
			r.cache.Add(destAddrPort, destAddrPort)
			logrus.Debugf("[%s] Not found User-Agent, Add LRU Relay Cache, Cache Len: %d",
				destAddrPort, r.cache.Len())
			if err := req.Write(dst); err != nil {
				logrus.Errorf("[%s][%s] write error: %s", destAddrPort, srcAddr, err.Error())
				return err
			}
			_, _ = io.Copy(dst, reader)
			return nil
		}

		isWhitelist := r.inWhitelist(originalUA)
		matches := true
		if r.pattern != "" {
			matches, err = r.uaRegex.MatchString(originalUA)
			if err != nil {
				logrus.Errorf("[%s][%s] User-Agent Regex Pattern Match Error: %s",
					destAddrPort, srcAddr, err.Error())
				matches = true
			}
		}

		// If UA is whitelisted or does not match target pattern, write once then pass-through.
		if isWhitelist || !matches {
			if !matches {
				logrus.Debugf("[%s][%s] Not Hit User-Agent Pattern: %s",
					destAddrPort, srcAddr, originalUA)
			}
			if isWhitelist {
				logrus.Debugf("[%s][%s] Hit User-Agent Whitelist: %s, Add LRU Relay Cache, Cache Len: %d",
					destAddrPort, srcAddr, originalUA, r.cache.Len())
				r.cache.Add(destAddrPort, destAddrPort)
			}
			statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
				Host: destAddrPort,
				UA:   originalUA,
			})
			if err := req.Write(dst); err != nil {
				logrus.Errorf("[%s][%s] write error: %s", destAddrPort, srcAddr, err.Error())
				return err
			}
			_, _ = io.Copy(dst, reader)
			return nil
		}

		// Rewrite UA and forward the request (including body)
		logrus.Debugf("[%s][%s] Hit User-Agent: %s", destAddrPort, srcAddr, originalUA)
		mockedUA := r.buildNewUA(originalUA)
		req.Header.Set("User-Agent", mockedUA)
		if err := req.Write(dst); err != nil {
			logrus.Errorf("[%s][%s] write error after replace user-agent: %s",
				destAddrPort, srcAddr, err.Error())
			return err
		}

		statistics.AddRewriteRecord(&statistics.RewriteRecord{
			Host:       destAddrPort,
			OriginalUA: originalUA,
			MockedUA:   mockedUA,
		})
	}
}

// isHTTP peeks the first few bytes and checks for a known HTTP method prefix.
func (r *Rewriter) isHTTP(reader *bufio.Reader) (bool, error) {
	buf, err := reader.Peek(7)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			logrus.Debugf("Peek EOF: %s", err.Error())
		} else {
			logrus.Errorf("Peek error: %s", err.Error())
		}
		return false, err
	}
	hint := string(buf)
	for _, m := range httpMethods {
		if strings.HasPrefix(hint, m) {
			return true, nil
		}
	}
	return false, nil
}

// buildNewUA returns either a partial replacement (regex) or full overwrite.
func (r *Rewriter) buildNewUA(originUA string) string {
	if r.enablePartialReplace && r.uaRegex != nil && r.pattern != "" {
		newUA, err := r.uaRegex.Replace(originUA, r.payloadUA, -1, -1)
		if err != nil {
			logrus.Errorf("User-Agent Replace Error: %s, use full overwrite", err.Error())
			return r.payloadUA
		}
		return newUA
	}
	return r.payloadUA
}

func (r *Rewriter) inWhitelist(ua string) bool {
	_, ok := r.whitelist[ua]
	return ok
}

func (r *Rewriter) logReadErr(destAddrPort string, src net.Conn, err error) {
	remote := src.RemoteAddr().String()
	switch {
	case errors.Is(err, io.EOF):
		logrus.Debugf("[%s][%s] read EOF in first phase", destAddrPort, remote)
	case strings.Contains(err.Error(), ErrUseClosedConn):
		logrus.Debugf("[%s][%s] read closed in first phase: %s", destAddrPort, remote, err.Error())
	case strings.Contains(err.Error(), ErrConnResetByPeer):
		logrus.Debugf("[%s][%s] read reset in first phase: %s", destAddrPort, remote, err.Error())
	case strings.Contains(err.Error(), ErrIOTimeout):
		logrus.Debugf("[%s][%s] read timeout in first phase: %s", destAddrPort, remote, err.Error())
	default:
		logrus.Errorf("[%s][%s] read error in first phase: %s", destAddrPort, remote, err.Error())
	}
}
