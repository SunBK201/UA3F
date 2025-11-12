package utils

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/sniff"
	"github.com/sunbk201/ua3f/internal/statistics"
)

var one = make([]byte, 1)

// Connect dials the target address and returns the connection.
func Connect(addr string) (target net.Conn, err error) {
	logrus.Debugf("Connecting %s", addr)
	if target, err = net.Dial("tcp", addr); err != nil {
		return nil, err
	}
	logrus.Debugf("Connected %s", addr)
	return target, nil
}

// CopyHalf copies from src to dst and half-closes both sides when done.
func CopyHalf(dst, src net.Conn) {
	defer func() {
		// Prefer TCP half-close to allow the opposite direction to drain.
		if tc, ok := dst.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = dst.Close()
		}
		if tc, ok := src.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = src.Close()
		}
		log.LogDebugWithAddr(src.RemoteAddr().String(), dst.RemoteAddr().String(), "Connections half-closed")
	}()
	io.CopyBuffer(dst, src, one)
}

// ProxyHalf runs the rewriter proxy on src->dst and then half-closes both sides.
func ProxyHalf(dst, src net.Conn, rw *rewrite.Rewriter, destAddr string) {
	srcAddr := src.RemoteAddr().String()

	defer func() {
		if tc, ok := dst.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = dst.Close()
		}
		if tc, ok := src.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = src.Close()
		}
		log.LogDebugWithAddr(src.RemoteAddr().String(), destAddr, "Connections half-closed")
		defer statistics.RemoveConnection(srcAddr, destAddr)
	}()

	statistics.AddConnection(&statistics.ConnectionRecord{
		Protocol:  sniff.TCP,
		SrcAddr:   srcAddr,
		DestAddr:  destAddr,
		StartTime: time.Now(),
	})

	if rw.Cache.Contains(destAddr) {
		// Fast path
		log.LogDebugWithAddr(srcAddr, destAddr, fmt.Sprintf("destination (%s) in cache, direct forward", destAddr))
		io.CopyBuffer(dst, src, one)
		return
	}
	_ = rw.Process(dst, src, destAddr, srcAddr)
}

func GetConnFD(conn net.Conn) (fd int, err error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, fmt.Errorf("not a TCP connection")
	}
	file, err := tcpConn.File()
	if err != nil {
		return 0, err
	}

	return int(file.Fd()), nil
}
