//go:build linux

package base

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const SO_MARK = 0xc9

// Connect dials the target address with SO_MARK set and returns the connection.
func Connect(addr string, mark int) (target net.Conn, err error) {
	dialer := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, mark)
			})
		},
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Connect dialer.Dial SO_MARK(%d): %v", mark, err)
	}
	return conn, nil
}

// GetOriginalDstAddr retrieves the original destination address of the redirected connection.
func GetOriginalDstAddr(conn net.Conn) (addr string, err error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", errors.New("GetConnFD connection is not *net.TCPConn")
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return "", fmt.Errorf("SyscallConn: %v", err)
	}

	var originalAddr string
	err = rawConn.Control(func(fd uintptr) {
		level := syscall.IPPROTO_IP
		if conn.RemoteAddr().String()[0] == '[' {
			level = syscall.IPPROTO_IPV6
		}

		addr, err := syscall.GetsockoptIPv6MTUInfo(int(fd), level, unix.SO_ORIGINAL_DST)
		if err != nil {
			slog.Warn("unix.GetsockoptIPv6MTUInfo", "error", err)
			return
		}

		var ip net.IP
		if level == syscall.IPPROTO_IPV6 {
			ip = net.IP(addr.Addr.Addr[:])
		} else {
			ipBytes := (*[4]byte)(unsafe.Pointer(&addr.Addr.Flowinfo))[:4]
			ip = net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
		}

		port := binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&addr.Addr.Port))[:2])

		if level == syscall.IPPROTO_IPV6 {
			originalAddr = fmt.Sprintf("[%s]:%d", ip.String(), port)
		} else {
			originalAddr = fmt.Sprintf("%s:%d", ip.String(), port)
		}
	})
	if err != nil {
		return "", fmt.Errorf("rawConn.Control: %v", err)
	}

	return originalAddr, nil
}
