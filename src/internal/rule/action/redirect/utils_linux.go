//go:build linux

package redirect

import (
	"net"
	"net/http"
	"syscall"
)

func sendRequest(req *http.Request) (*http.Response, error) {
	dialer := &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(
					int(fd),
					syscall.SOL_SOCKET,
					syscall.SO_MARK,
					0xc9,
				)
			})
		},
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}

	resp, err := client.Do(req)
	return resp, err
}
