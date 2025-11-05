package sniff

import (
	"bufio"
	"strings"
)

// HTTP methods used to detect HTTP by request line.
var methodBytes = [...][]byte{
	[]byte("GET"),
	[]byte("POST"),
	[]byte("HEAD"),
	[]byte("CONNECT"),
	[]byte("PUT"),
	[]byte("DELETE"),
	[]byte("OPTIONS"),
	[]byte("PATCH"),
	[]byte("TRACE"),
}

const maxMethodLen = 7

type Node struct {
	next map[byte]*Node
	end  bool
}

var root *Node

func init() {
	root = &Node{next: make(map[byte]*Node)}
	for _, m := range methodBytes {
		node := root
		for _, c := range m {
			if node.next[c] == nil {
				node.next[c] = &Node{next: make(map[byte]*Node)}
			}
			node = node.next[c]
		}
		node.end = true
	}
}

// beginWithHTTPMethod peeks the first few bytes to check for known HTTP method prefixes.
func beginWithHTTPMethod(reader *bufio.Reader) (bool, error) {
	node := root
	var prevLen int

	for n := 3; n <= maxMethodLen; n++ {
		buf, err := reader.Peek(n)
		if err != nil {
			return false, err
		}
		for i := prevLen; i < len(buf); i++ {
			c := buf[i]
			next, ok := node.next[c]
			if !ok {
				return false, nil
			}
			node = next
			if node.end {
				return true, nil
			}
		}
		prevLen = len(buf)
	}

	return false, nil
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
