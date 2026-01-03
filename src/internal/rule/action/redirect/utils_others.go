//go:build !linux

package redirect

import (
	"net/http"
)

func sendRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}
