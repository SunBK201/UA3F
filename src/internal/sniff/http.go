package sniff

import (
	"bufio"
	"strings"
	"time"
)

// HTTP methods used to detect HTTP by request line.
var methods = [...]string{"GET", "POST", "HEAD", "CONNECT", "PUT", "DELETE", "OPTIONS", "PATCH", "TRACE"}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	method, rest, ok1 := strings.Cut(line, " ")
	requestURI, proto, ok2 := strings.Cut(rest, " ")
	if !ok1 || !ok2 {
		return "", "", "", false
	}
	return method, requestURI, proto, true
}

// beginWithHTTPMethod peeks the first few bytes to check for known HTTP method prefixes.
func beginWithHTTPMethod(reader *bufio.Reader) (bool, error) {
	const maxMethodLen = 7
	const minMethodLen = 3
	var hint []byte
	hint, err := PeekWithTimeout(reader, maxMethodLen, 3*time.Second)
	if err != nil {
		if err != ErrPeekTimeout {
			return false, err
		}
		hint, err = PeekWithTimeout(reader, minMethodLen+1, time.Second)
		if err != nil {
			return false, err
		}
	}
	method, _, _ := strings.Cut(string(hint), " ")
	for _, m := range methods {
		if method == m {
			return true, nil
		}
	}
	return false, nil
}

// SniffHTTP peeks the first few bytes and checks for a known HTTP method prefix.
func SniffHTTP(reader *bufio.Reader) (bool, error) {
	// Fast check: peek first word to see if it's a known HTTP method
	beginHTTP, err := beginWithHTTPMethod(reader)
	if err != nil {
		return false, err
	}

	// Detailed check: parse request line
	line, err := peekLineString(reader, 128)
	if err != nil {
		return beginHTTP, nil
	}
	_, _, proto, ok := parseRequestLine(line)
	if !ok {
		return beginHTTP, nil
	}
	if proto != "HTTP/1.1" && proto != "HTTP/1.0" {
		return false, nil
	}
	return beginHTTP, nil
}

func SniffHTTPFast(reader *bufio.Reader) (bool, error) {
	beginHTTP, err := beginWithHTTPMethod(reader)
	if err != nil {
		return false, err
	}
	return beginHTTP, nil
}
