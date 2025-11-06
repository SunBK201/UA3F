package sniff

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Protocol sniffed protocol types
type Protocol string

const (
	TCP       Protocol = "TCP"
	HTTP      Protocol = "HTTP"
	HTTPS     Protocol = "HTTPS"
	TLS       Protocol = "TLS"
	WebSocket Protocol = "WebSocket"
)

var ErrPeekTimeout = errors.New("peek timeout")

// peekLineSlice reads a line from bufio.Reader without consuming it.
// returns the line bytes (without CRLF) or error.
func peekLineSlice(br *bufio.Reader, maxSize int) ([]byte, error) {
	var line []byte

	peekSize := maxSize
	if peekSize == 0 {
		return nil, io.EOF
	}
	if buffered := br.Buffered(); buffered < peekSize {
		peekSize = buffered
	}

	buf, err := br.Peek(peekSize)
	if err != nil {
		return nil, err
	}

	if i := bytes.IndexByte(buf, '\n'); i >= 0 {
		line = append(line, buf[:i]...)
		// Remove trailing CR if present
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		return line, nil
	}
	return nil, io.EOF
}

// peekLineString reads a line from bufio.Reader without consuming it.
// returns the line string (without CRLF) or error.
func peekLineString(br *bufio.Reader, maxSize int) (string, error) {
	lineBytes, err := peekLineSlice(br, maxSize)
	if err != nil {
		return "", err
	}
	return string(lineBytes), nil
}
