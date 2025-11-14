package rewrite

import (
	"fmt"

	"github.com/google/gopacket/layers"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// RewriteResult contains the result of TCP rewriting
type RewriteResult struct {
	Modified bool // Whether the packet was modified
	HasUA    bool // Whether User-Agent was found
	InCache  bool // Whether destination address is in cache
}

// shouldRewriteUA checks if the given User-Agent should be rewritten
// Returns true if UA should be rewritten (not in whitelist and matches regex pattern)
func (r *Rewriter) shouldRewriteUA(srcAddr, dstAddr string, ua string) bool {
	// If no pattern specified, rewrite all non-whitelist UAs
	if r.pattern == "" {
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

// buildReplacement creates replacement content for User-Agent
// If the original UA should not be rewritten, returns nil
// Otherwise, uses buildUserAgent logic (partial or full replace) and adjusts to length n
func (r *Rewriter) buildReplacement(srcAddr, dstAddr string, originalUA string, n int) []byte {
	if n <= 0 {
		return nil
	}

	// Build the new UA using the same logic as in Rewrite()
	newUA := r.buildUserAgent(originalUA)

	log.LogInfoWithAddr(srcAddr, dstAddr, fmt.Sprintf("Rewritten User-Agent: %s", newUA))
	statistics.AddRewriteRecord(&statistics.RewriteRecord{
		Host:       dstAddr,
		OriginalUA: originalUA,
		MockedUA:   newUA,
	})

	// Adjust to the exact length needed
	if len(newUA) >= n {
		return []byte(newUA[:n])
	}
	out := make([]byte, n)
	copy(out, newUA)
	// Pad with spaces if newUA is shorter than needed
	for i := len(newUA); i < n; i++ {
		out[i] = ' '
	}

	return out
}

// RewritePacketUserAgent rewrites User-Agent in a raw packet payload in-place
// Returns metadata about the operation
func (r *Rewriter) RewritePacketUserAgent(payload []byte, srcAddr, dstAddr string) (hasUA, modified bool) {
	// Find all User-Agent positions
	positions, unterm := findUserAgentInPayload(payload)

	if unterm {
		log.LogInfoWithAddr(srcAddr, dstAddr, "Unterminated User-Agent found, not rewriting")
		return true, false
	}

	if len(positions) == 0 {
		log.LogDebugWithAddr(srcAddr, dstAddr, "No User-Agent found in payload")
		return false, false
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

		// Check if should rewrite
		if !r.shouldRewriteUA(srcAddr, dstAddr, originalUA) {
			statistics.AddPassThroughRecord(&statistics.PassThroughRecord{
				SrcAddr:  srcAddr,
				DestAddr: dstAddr,
				UA:       originalUA,
			})
			return true, false
		}

		// Build replacement with regex matching
		repl := r.buildReplacement(srcAddr, dstAddr, originalUA, n)
		if repl != nil {
			copy(payload[valStart:valEnd], repl)
			modified = true
		}
	}
	return true, modified
}

// RewriteTCP rewrites the TCP packet's User-Agent if applicable
func (r *Rewriter) RewriteTCP(tcp *layers.TCP, srcAddr, dstAddr string) *RewriteResult {
	if len(tcp.Payload) == 0 {
		log.LogDebugWithAddr(srcAddr, dstAddr, "TCP payload is empty")
		return &RewriteResult{Modified: false}
	}

	hasUA, modified := r.RewritePacketUserAgent(tcp.Payload, srcAddr, dstAddr)
	return &RewriteResult{
		Modified: modified,
		HasUA:    hasUA,
	}
}
