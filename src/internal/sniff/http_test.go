package sniff

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestBeginWithHTTPMethod(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMatch bool
		wantErr   bool
	}{
		{"GET method", "GET /index.html HTTP/1.1\r\n", true, false},
		{"POST method", "POST /submit HTTP/1.1\r\n", true, false},
		{"HEAD method", "HEAD /abc HTTP/1.0\r\n", true, false},
		{"PUT method", "PUT /resource HTTP/1.1\r\n", true, false},
		{"PATCH method", "PATCH /item HTTP/1.1\r\n", true, false},
		{"OPTIONS method", "OPTIONS * HTTP/1.1\r\n", true, false},
		{"TRACE method", "TRACE / HTTP/1.1\r\n", true, false},
		{"CONNECT method", "CONNECT example.com:443 HTTP/1.1\r\n", true, false},
		{"DELETE method", "DELETE /resource HTTP/1.1\r\n", true, false},
		{"lowercase method", "get /index.html HTTP/1.1\r\n", false, false},
		{"non-http prefix", "HELLO WORLD", false, false},
		{"empty input", "", false, true}, // peek 会返回错误
		{"short incomplete input", "GE", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(bytes.NewBufferString(tt.input))
			got, err := beginWithHTTPMethod(reader)

			if (err != nil) != tt.wantErr {
				// differentiate between EOF (expected) and other errors sometimes
				if !tt.wantErr || !errors.Is(err, io.EOF) {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if got != tt.wantMatch {
				t.Errorf("beginWithHTTPMethod(%q) = %v, want %v", tt.input, got, tt.wantMatch)
			}
		})
	}
}
