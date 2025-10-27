package http

import (
	"bytes"
	"errors"
	"strconv"
)

type header struct {
	Name        []byte
	Value       []byte
	StartOffset int
	EndOffset   int
}

type HTTPParser struct {
	Method, Path, Version []byte

	Headers      []header
	TotalHeaders int

	host     []byte
	hostRead bool

	contentLength     int64
	contentLengthRead bool
}

const DefaultHeaderSlice = 10

// Create a new parser
func NewHTTPParser() *HTTPParser {
	return NewSizedHTTPParser(DefaultHeaderSlice)
}

// Create a new parser allocating size for size headers
func NewSizedHTTPParser(size int) *HTTPParser {
	return &HTTPParser{
		Headers:       make([]header, size),
		TotalHeaders:  size,
		contentLength: -1,
	}
}

var (
	ErrBadProto    = errors.New("bad protocol")
	ErrMissingData = errors.New("missing data")
	ErrUnsupported = errors.New("unsupported http feature")
)

const (
	eNextHeader int = iota
	eNextHeaderN
	eHeader
	eHeaderValueSpace
	eHeaderValue
	eHeaderValueN
	eMLHeaderStart
	eMLHeaderValue
)

// Parse the buffer as an HTTP Request. The buffer must contain the entire
// request or Parse will return ErrMissingData for the caller to get more
// data. (this thusly favors getting a completed request in a single Read()
// call).
//
// Returns the number of bytes used by the header (thus where the body begins).
// Also can return ErrUnsupported if an HTTP feature is detected but not supported.
func (hp *HTTPParser) Parse(input []byte) (int, error) {
	var headers int
	var path int
	var ok bool

	total := len(input)

method:
	for i := 0; i < total; i++ {
		switch input[i] {
		case ' ', '\t':
			hp.Method = input[0:i]
			ok = true
			path = i + 1
			break method
		}
	}

	if !ok {
		return 0, ErrMissingData
	}

	var version int

	ok = false

path:
	for i := path; i < total; i++ {
		switch input[i] {
		case ' ', '\t':
			ok = true
			hp.Path = input[path:i]
			version = i + 1
			break path
		}
	}

	if !ok {
		return 0, ErrMissingData
	}

	var readN bool

	ok = false
loop:
	for i := version; i < total; i++ {
		c := input[i]

		switch readN {
		case false:
			switch c {
			case '\r':
				hp.Version = input[version:i]
				readN = true
			case '\n':
				hp.Version = input[version:i]
				headers = i + 1
				ok = true
				break loop
			}
		case true:
			if c != '\n' {
				return 0, errors.New("missing newline in version")
			}
			headers = i + 1
			ok = true
			break loop
		}
	}

	if !ok {
		return 0, ErrMissingData
	}

	var h int

	var headerName []byte

	state := eNextHeader

	start := headers

	for i := headers; i < total; i++ {
		switch state {
		case eNextHeader:
			switch input[i] {
			case '\r':
				state = eNextHeaderN
			case '\n':
				return i + 1, nil
			case ' ', '\t':
				state = eMLHeaderStart
			default:
				start = i
				state = eHeader
			}
		case eNextHeaderN:
			if input[i] != '\n' {
				return 0, ErrBadProto
			}

			return i + 1, nil
		case eHeader:
			if input[i] == ':' {
				headerName = input[start:i]
				state = eHeaderValueSpace
			}
		case eHeaderValueSpace:
			switch input[i] {
			case ' ', '\t':
				continue
			}

			start = i
			state = eHeaderValue
		case eHeaderValue:
			switch input[i] {
			case '\r':
				state = eHeaderValueN
			case '\n':
				state = eNextHeader
			default:
				continue
			}

			hp.Headers[h] = header{headerName, input[start:i], start, i}
			h++

			if h == hp.TotalHeaders {
				newHeaders := make([]header, hp.TotalHeaders+10)
				copy(newHeaders, hp.Headers)
				hp.Headers = newHeaders
				hp.TotalHeaders += 10
			}
		case eHeaderValueN:
			if input[i] != '\n' {
				return 0, ErrBadProto
			}
			state = eNextHeader

		case eMLHeaderStart:
			switch input[i] {
			case ' ', '\t':
				continue
			}

			start = i
			state = eMLHeaderValue
		case eMLHeaderValue:
			switch input[i] {
			case '\r':
				state = eHeaderValueN
			case '\n':
				state = eNextHeader
			default:
				continue
			}

			cur := hp.Headers[h-1].Value

			newheader := make([]byte, len(cur)+1+(i-start))
			copy(newheader, cur)
			copy(newheader[len(cur):], []byte(" "))
			copy(newheader[len(cur)+1:], input[start:i])

			hp.Headers[h-1].Value = newheader
		}
	}

	return 0, ErrMissingData
}

// Return a value of a header matching name.
func (hp *HTTPParser) FindHeader(name []byte) (value []byte, startOffset, endOffset int) {
	for _, header := range hp.Headers {
		if bytes.Equal(header.Name, name) {
			return header.Value, header.StartOffset, header.EndOffset
		}
	}

	for _, header := range hp.Headers {
		if bytes.EqualFold(header.Name, name) {
			return header.Value, header.StartOffset, header.EndOffset
		}
	}

	return nil, 0, 0
}

var cContentLength = []byte("Content-Length")

// Return the value of the Content-Length header.
// A value of -1 indicates the header was not set.
func (hp *HTTPParser) ContentLength() int64 {
	if hp.contentLengthRead {
		return hp.contentLength
	}

	header, _, _ := hp.FindHeader(cContentLength)
	if header != nil {
		i, err := strconv.ParseInt(string(header), 10, 0)
		if err == nil {
			hp.contentLength = i
		}
	}

	hp.contentLengthRead = true
	return hp.contentLength
}

// Return the value of the Host header
func (hp *HTTPParser) Host() []byte {
	if hp.hostRead {
		return hp.host
	}

	hp.hostRead = true
	hp.host, _, _ = hp.FindHeader([]byte("Host"))
	return hp.host
}
