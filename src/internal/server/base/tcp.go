//go:build !linux

package base

import (
	"fmt"
	"net"
)

func Connect(addr string, mark int) (target net.Conn, err error) {
	if target, err = net.Dial("tcp", addr); err != nil {
		return nil, fmt.Errorf("net.Dial: %v", err)
	}
	return target, nil
}
