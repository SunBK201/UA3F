package rewrite

import (
	"bytes"
)

var (
	// HTTP User-Agent header tag (case-sensitive search optimized)
	uaTag = []byte("\r\nUser-Agent:")
)

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
