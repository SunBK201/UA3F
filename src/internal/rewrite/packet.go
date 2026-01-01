package rewrite

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type PacketRewriter struct {
	rewriteMode    config.RewriteMode
	UserAgent      string
	uaRegex        *regexp2.Regexp
	partialReplace bool
	Recorder       *statistics.Recorder
}

var (
	uaTag = []byte("\r\nUser-Agent:")
)

func (r *PacketRewriter) RewriteRequest(metadata *common.Metadata) (decision *RewriteDecision) {
	if r.rewriteMode == config.RewriteModeDirect {
		return &RewriteDecision{
			Modified: false,
		}
	}
	if len(metadata.Packet.TCP.Payload) == 0 {
		return &RewriteDecision{
			Modified: false,
		}
	}
	hasUA, modified, skip := r.RewritePacketUserAgent(metadata.Packet.TCP.Payload, metadata.SrcAddr(), metadata.DestAddr())
	return &RewriteDecision{
		Modified: modified,
		HasUA:    hasUA,
		NeedSkip: skip,
	}
}

func (r *PacketRewriter) RewriteResponse(metadata *common.Metadata) (decision *RewriteDecision) {
	return &RewriteDecision{
		Action: action.DirectAction,
	}
}

func (r *PacketRewriter) ServeRequest() bool {
	return true
}

func (r *PacketRewriter) ServeResponse() bool {
	return false
}

func NewPacketRewriter(cfg *config.Config, recorder *statistics.Recorder) (*PacketRewriter, error) {
	var regex *regexp2.Regexp
	var err error
	if cfg.UserAgentRegex != "" {
		regex, err = regexp2.Compile("(?i)"+cfg.UserAgentRegex, regexp2.None)
		if err != nil {
			return nil, err
		}
	}
	return &PacketRewriter{
		rewriteMode:    cfg.RewriteMode,
		UserAgent:      cfg.UserAgent,
		uaRegex:        regex,
		partialReplace: cfg.UserAgentPartialReplace,
		Recorder:       recorder,
	}, nil
}

// shouldRewriteUA determines if the User-Agent should be rewritten
func (r *PacketRewriter) shouldRewriteUA(srcAddr, dstAddr string, ua string) bool {
	if r.uaRegex == nil {
		return true
	}

	// Check regex match
	matches, err := r.uaRegex.MatchString(ua)
	if err != nil {
		log.LogErrorWithAddr(srcAddr, dstAddr, fmt.Sprintf("r.uaRegex.MatchString Error matching User-Agent regex: %v", err))
		return true
	}

	return matches
}

// buildUserAgent returns either a partial replacement (regex) or full overwrite.
func (r *PacketRewriter) buildUserAgent(originUA string) string {
	if r.partialReplace && r.uaRegex != nil {
		newUA, err := r.uaRegex.Replace(originUA, r.UserAgent, -1, -1)
		if err != nil {
			slog.Error("r.uaRegex.Replace", slog.Any("error", err))
			return r.UserAgent
		}
		return newUA
	}
	return r.UserAgent
}

// buildReplacement creates replacement content for User-Agent
// If the original UA should not be rewritten, returns nil
// Otherwise, uses buildUserAgent logic (partial or full replace) and adjusts to length n
func (r *PacketRewriter) buildReplacement(srcAddr, dstAddr string, originalUA string, n int) []byte {
	if n <= 0 {
		return nil
	}

	// Build the new UA using the same logic as in Rewrite()
	newUA := r.buildUserAgent(originalUA)

	log.LogInfoWithAddr(srcAddr, dstAddr, fmt.Sprintf("Rewritten User-Agent: %s", newUA))
	r.Recorder.AddRecord(&statistics.RewriteRecord{
		Host:       dstAddr,
		OriginalUA: originalUA,
		MockedUA:   newUA,
	})

	// Adjust to the exact length needed
	newUALen := len(newUA)
	if newUALen >= n {
		return []byte(newUA[:n])
	}
	out := make([]byte, n)
	copy(out, newUA)
	// Pad with spaces if newUA is shorter than needed
	for i := newUALen; i < n; i++ {
		out[i] = ' '
	}

	return out
}

// RewritePacketUserAgent rewrites User-Agent in a raw packet payload in-place
// Returns metadata about the operation
func (r *PacketRewriter) RewritePacketUserAgent(payload []byte, srcAddr, dstAddr string) (hasUA, modified, skip bool) {
	// Find all User-Agent positions
	positions, unterm := findUserAgentInPayload(payload)

	if unterm {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Unterminated User-Agent found, not rewriting")
		return true, false, false
	}

	if len(positions) == 0 {
		log.LogDebugWithAddr(srcAddr, dstAddr, "No User-Agent found in payload")
		return false, false, false
	}

	// Replace each User-Agent value in-place
	for _, pos := range positions {
		valStart, valEnd := pos[0], pos[1]
		n := valEnd - valStart
		if n <= 0 {
			continue
		}

		// Extract original UA string
		originalUA := string(payload[valStart:valEnd])

		log.LogInfoWithAddr(srcAddr, dstAddr, fmt.Sprintf("Original User-Agent: %s", originalUA))

		if originalUA == "Valve/Steam HTTP Client 1.0" {
			return true, false, true
		}

		// Check if should rewrite
		if !r.shouldRewriteUA(srcAddr, dstAddr, originalUA) {
			r.Recorder.AddRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: dstAddr,
				UA:       originalUA,
			})
			return true, false, false
		}

		// Build replacement with regex matching
		repl := r.buildReplacement(srcAddr, dstAddr, originalUA, n)
		if repl != nil {
			copy(payload[valStart:valEnd], repl)
			modified = true
		}
	}
	return true, modified, false
}

// toLowerASCII converts an ASCII byte to lowercase (only A-Z)
func toLowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// indexFoldASCII performs case-insensitive search for needle in haystack (ASCII only)
// Returns the first occurrence index or -1 if not found
func indexFoldASCII(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	if len(haystack) < len(needle) {
		return -1
	}
	n0 := toLowerASCII(needle[0])
	limit := len(haystack) - len(needle)
	for i := 0; i <= limit; i++ {
		if toLowerASCII(haystack[i]) != n0 {
			continue
		}
		match := true
		for j := 1; j < len(needle); j++ {
			if toLowerASCII(haystack[i+j]) != toLowerASCII(needle[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// findUserAgentInPayload searches for User-Agent header(s) in raw HTTP payload
// Returns slice of (startPos, endPos) pairs for each User-Agent value found
// Returns empty slice if no User-Agent found, or if any UA is unterminated (missing \r)
func findUserAgentInPayload(payload []byte) (positions [][2]int, unterminated bool) {
	if len(payload) < len(uaTag) {
		return nil, false
	}

	searchStart := 0
	for searchStart <= len(payload)-len(uaTag) {

		idx := indexFoldASCII(payload[searchStart:], uaTag)
		if idx < 0 {
			break
		}

		uaKeyPos := searchStart + idx
		valStart := uaKeyPos + len(uaTag)

		// Support both "User-Agent:XXX" and "User-Agent: XXX" (with or without space)
		if valStart < len(payload) && payload[valStart] == ' ' {
			valStart++
		}
		if valStart >= len(payload) {
			// UA at the end of payload, no \r found
			return nil, true
		}

		// Find line ending position: look for \r
		relEnd := bytes.IndexByte(payload[valStart:], '\r')
		if relEnd < 0 {
			// No \r found, UA is unterminated
			return nil, true
		}
		valEnd := valStart + relEnd

		if valEnd > valStart {
			positions = append(positions, [2]int{valStart, valEnd})
		}

		// Continue searching for more UA headers
		searchStart = valEnd
	}

	return positions, false
}
