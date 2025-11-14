//go:build linux

package base

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

const SO_MARK = 0xc9

// ConnectWithMark dials the target address with SO_MARK set and returns the connection.
func ConnectWithMark(addr string, mark int) (target net.Conn, err error) {
	dialer := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, mark)
			})
		},
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ConnectWithMark dialer.Dial SO_MARK(%d): %v", mark, err)
	}
	return conn, nil
}

// GetOriginalDstAddr retrieves the original destination address of the redirected connection.
func GetOriginalDstAddr(conn net.Conn) (addr string, err error) {
	fd, err := GetConnFD(conn)
	if err != nil {
		return "", fmt.Errorf("GetConnFD: %v", err)
	}
	raw, err := unix.GetsockoptIPv6Mreq(fd, unix.SOL_IP, unix.SO_ORIGINAL_DST)
	if err != nil {
		return "", fmt.Errorf("unix.GetsockoptIPv6Mreq: %v", err)
	}

	ip := net.IPv4(raw.Multiaddr[4], raw.Multiaddr[5], raw.Multiaddr[6], raw.Multiaddr[7])
	port := uint16(raw.Multiaddr[2])<<8 + uint16(raw.Multiaddr[3])
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}
