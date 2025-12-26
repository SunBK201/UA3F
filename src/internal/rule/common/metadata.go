package common

import "net/http"

type Metadata struct {
	Request  *http.Request
	SrcAddr  string
	DestAddr string
}
