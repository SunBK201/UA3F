package rewrite

import (
	"bufio"
	"bytes"
	"errors"
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
	"github.com/sunbk201/ua3f/internal/statistics"
)

const (
	ErrUseClosedConn   = "use of closed network connection"
	ErrConnResetByPeer = "connection reset by peer"
	ErrIOTimeout       = "i/o timeout"
)

// HTTP methods used to detect HTTP by request line.
var httpMethods = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"HEAD":    {},
	"PUT":     {},
	"PATCH":   {},
	"DELETE":  {},
	"OPTIONS": {},
	"TRACE":   {},
	"CONNECT": {},
}

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
func (r *Rewriter) ProxyHTTPOrRaw(dst net.Conn, src net.Conn, destAddr string) (err error) {
	srcAddr := src.RemoteAddr().String()

	// Fast path: known pass-through
	if r.cache.Contains(destAddr) {
		log.LogDebugWithAddr(src.RemoteAddr().String(), destAddr, "LRU Relay Cache Hit, pass-through")
		io.Copy(dst, src)
		return nil
	}

	reader := bufio.NewReader(src)
	defer func() {
		if err != nil {
			log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("ProxyHTTPOrRaw Error: %s", err.Error()))
		}
		io.Copy(dst, reader)
	}()

	isHTTP, err := r.isHTTP(reader)
	if err != nil {
		err = fmt.Errorf("isHTTP: %w", err)
		return
	}
	if !isHTTP {
		r.cache.Add(destAddr, destAddr)
		log.LogDebugWithAddr(srcAddr, destAddr, "Not HTTP, added to LRU Relay Cache")
		return
	}

	var req *http.Request

	// HTTP request loop (handles keep-alive)
	for {
		isHTTP, err = r.isHTTP(reader)
		if err != nil {
			err = fmt.Errorf("isHTTP: %w", err)
			return
		}
		if !isHTTP {
			h2, _ := reader.Peek(2) // ensure we have at least 2 bytes
			if isWebSocket(h2) {
				log.LogDebugWithAddr(srcAddr, destAddr, "WebSocket detected, pass-through")
			} else {
				r.cache.Add(destAddr, destAddr)
				log.LogDebugWithAddr(srcAddr, destAddr, "Not HTTP, added to LRU Relay Cache")
			}
			return
		}
		req, err = http.ReadRequest(reader)
		if err != nil {
			err = fmt.Errorf("http.ReadRequest: %w", err)
			return
		}

		originalUA := req.Header.Get("User-Agent")

		// No UA header: pass-through after writing this first request
		if originalUA == "" {
			r.cache.Add(destAddr, destAddr)
			log.LogDebugWithAddr(srcAddr, destAddr, "Not found User-Agent, Add LRU Relay Cache")
			if err = req.Write(dst); err != nil {
				err = fmt.Errorf("req.Write: %w", err)
			}
			return
		}

		isWhitelist := r.inWhitelist(originalUA)
		matches := true
		if r.pattern != "" {
			matches, err = r.uaRegex.MatchString(originalUA)
			if err != nil {
				log.LogErrorWithAddr(srcAddr, destAddr, fmt.Sprintf("User-Agent Regex Pattern Match Error: %s", err.Error()))
				matches = true
			}
		}

		// If UA is whitelisted or does not match target pattern, write once then pass-through.
		if isWhitelist || !matches {
			if !matches {
				log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Not Hit User-Agent Regex: %s", originalUA))
			}
			if isWhitelist {
				log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Hit User-Agent Whitelist: %s", originalUA))
				r.cache.Add(destAddr, destAddr)
			}
			statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
				Host: destAddr,
				UA:   originalUA,
			})
			if err = req.Write(dst); err != nil {
				err = fmt.Errorf("req.Write: %w", err)
			}
			return
		}

		// Rewrite UA and forward the request (including body)
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("Hit User-Agent: %s", originalUA))
		mockedUA := r.buildNewUA(originalUA)
		req.Header.Set("User-Agent", mockedUA)
		if err = req.Write(dst); err != nil {
			err = fmt.Errorf("req.Write: %w", err)
			return
		}

		statistics.AddRewriteRecord(&statistics.RewriteRecord{
			Host:       destAddr,
			OriginalUA: originalUA,
			MockedUA:   mockedUA,
		})
	}
}

// isHTTP peeks the first few bytes and checks for a known HTTP method prefix.
func (r *Rewriter) isHTTP(reader *bufio.Reader) (bool, error) {
	// Fast check: peek first word to see if it's a known HTTP method
	const maxMethodLen = 7
	hintSlice, err := reader.Peek(maxMethodLen)
	if err != nil {
		return false, err
	}
	hint := string(hintSlice)
	method, _, _ := strings.Cut(hint, " ")
	if _, exists := httpMethods[method]; !exists {
		return false, nil
	}

	// Detailed check: parse request line
	line, err := peekLineString(reader)
	if err != nil {
		return false, err
	}
	method, _, proto, ok := parseRequestLine(line)
	if !ok {
		return false, nil
	}
	if proto != "HTTP/1.1" && proto != "HTTP/1.0" {
		return false, nil
	}
	if _, exists := httpMethods[method]; exists {
		return true, nil
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

// peekLineSlice reads a line from bufio.Reader without consuming it.
// returns the line bytes (without CRLF) or error.
func peekLineSlice(br *bufio.Reader) ([]byte, error) {
	const chunkSize = 256
	var line []byte

	offset := 0
	for {
		// Ensure there is data in the buffer
		n := br.Buffered()
		if n == 0 {
			// No data in buffer, try to fill it
			_, err := br.Peek(1)
			if err != nil {
				return nil, err
			}
			n = br.Buffered()
		}

		// Limit to chunkSize
		if n > chunkSize {
			n = chunkSize
		}

		buf, err := br.Peek(offset + n)
		if err != nil && !errors.Is(err, bufio.ErrBufferFull) && !errors.Is(err, io.EOF) {
			return nil, err
		}

		data := buf[offset:]
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line = append(line, data[:i]...)
			// Remove trailing CR if present
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			return line, nil
		}

		line = append(line, data...)
		offset += len(data)

		// If EOF reached without finding a newline, return what we have
		if errors.Is(err, io.EOF) {
			return line, io.EOF
		}
	}
}

// peekLineString reads a line from bufio.Reader without consuming it.
// returns the line string (without CRLF) or error.
func peekLineString(br *bufio.Reader) (string, error) {
	lineBytes, err := peekLineSlice(br)
	if err != nil {
		return "", err
	}
	return string(lineBytes), nil
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	method, rest, ok1 := strings.Cut(line, " ")
	requestURI, proto, ok2 := strings.Cut(rest, " ")
	if !ok1 || !ok2 {
		return "", "", "", false
	}
	return method, requestURI, proto, true
}

func isWebSocket(header []byte) bool {
	if len(header) < 2 {
		return false
	}

	b0 := header[0]
	b1 := header[1]

	rsv := b0 & 0x70    // RSV1-3
	opcode := b0 & 0x0F // opcode
	mask := b1 & 0x80   // MASK

	// requested frames from client to server must be masked
	if mask == 0 {
		return false
	}
	// Control frames must have FIN set
	if rsv != 0 {
		return false
	}
	// opcode must be in valid range
	if opcode > 0xA {
		return false
	}
	// payload length
	payloadLen := b1 & 0x7F
	if payloadLen > 0 && payloadLen <= 125 {
		return true
	}

	return true
}
