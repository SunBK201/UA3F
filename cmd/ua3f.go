package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"
	"time"
	"ua3f/http"
	"ua3f/log"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
)

var version = "0.2.2"
var payloadByte []byte
var cache *expirable.LRU[string, string]
var HTTP_METHOD = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE", "CONNECT"}
var whitelist = []string{
	"MicroMessenger Client",
	"ByteDancePcdn",
	"Go-http-client/1.1",
}

const RDBUF = 1024 * 8

// var dpool *ants.PoolWithFunc
// var gpool *ants.PoolWithFunc
//
// type RelayConn struct {
// 	src          net.Conn
// 	dst          net.Conn
// 	destAddrPort string
// }

func main() {
	var payload string
	var addr string
	var port int
	var loglevel string

	flag.StringVar(&addr, "b", "127.0.0.1", "bind address (default: 127.0.0.1)")
	flag.IntVar(&port, "p", 1080, "port")
	flag.StringVar(&payload, "f", "FFF", "User-Agent")
	flag.StringVar(&loglevel, "l", "info", "Log level (default: info)")
	flag.Parse()

	log.SetLogConf(loglevel)

	logrus.Info("UA3F v" + version)
	logrus.Info(fmt.Sprintf("Port: %d", port))
	logrus.Info(fmt.Sprintf("User-Agent: %s", payload))
	logrus.Info(fmt.Sprintf("Log level: %s", loglevel))

	cache = expirable.NewLRU[string, string](300, nil, time.Second*600)

	// dpool, _ = ants.NewPoolWithFunc(1000, forward)
	// gpool, _ = ants.NewPoolWithFunc(500, gforward)
	// defer dpool.Release()
	// defer gpool.Release()

	payloadByte = []byte(payload)

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		logrus.Fatal("Listen failed: ", err)
		return
	}
	logrus.Info(fmt.Sprintf("Listen on %s:%d", addr, port))
	for {
		client, err := server.Accept()
		if err != nil {
			logrus.Error("Accept failed: ", err)
			continue
		}
		logrus.Debug(fmt.Sprintf("Accept %s", client.RemoteAddr().String()))
		go process(client)
	}
}

func process(client net.Conn) {
	if err := Socks5Auth(client); err != nil {
		// logrus.Error("Auth failed: ", err)
		client.Close()
		return
	}
	target, destAddrPort, err := Socks5Connect(client)
	if err != nil {
		logrus.Error("Connect failed: ", err)
		client.Close()
		return
	}
	Socks5Forward(client, target, destAddrPort)
	// Socks5Relay(client, target, destAddrPort)
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		if err == io.EOF {
			logrus.Warn(fmt.Sprintf("[%s][Auth] read EOF", client.RemoteAddr().String()))
		} else {
			logrus.Error(fmt.Sprintf("[%s][Auth] read header: %s", client.RemoteAddr().String(), err.Error()))
		}
		return errors.New("reading header:" + err.Error())
	}
	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		logrus.Error(fmt.Sprintf("[%s][Auth] invalid ver", client.RemoteAddr().String()))
		return errors.New("invalid version")
	}
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		logrus.Error(fmt.Sprintf("[%s][Auth] read methods: %s", client.RemoteAddr().String(), err.Error()))
		return errors.New("read methods:" + err.Error())
	}
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		logrus.Error(fmt.Sprintf("[%s][Auth] write rsp: %s", client.RemoteAddr().String(), err.Error()))
		return errors.New("write rsp:" + err.Error())
	}
	return nil
}

// func Socks5UDP() {
//	https://datatracker.ietf.org/doc/html/rfc1928
// 	// _, _ = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x7f, 0, 0, 0x1, 0x04, 0x38})
// 	_, _ = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0x04, 0x38})
// 	server, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 1080})
// 	_, _ = server.Read(buf[:4])
// 	frag, atyp := buf[2], buf[3]
// 	addr := ""
// 	switch atyp {
//	case 1:
//		n, err = server.Read(buf[:4])
//		if n != 4 {
//			return nil, "", errors.New("invalid IPv4:" + err.Error())
//		}
//		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
//	case 3:
//		n, err = server.Read(buf[:1])
//		if n != 1 {
//			return nil, "", errors.New("invalid hostname:" + err.Error())
//		}
//		addrLen := int(buf[0])
//		n, err = server.Read(buf[:addrLen])
//		if n != addrLen {
//			return nil, "", errors.New("invalid hostname:" + err.Error())
//		}
//		addr = string(buf[:addrLen])
//	case 4:
//		return nil, "", errors.New("IPv6: no supported yet")
//	default:
//		return nil, "", errors.New("invalid atyp")
// 	}
// 	n, err = server.Read(buf[:2])
// 	port := binary.BigEndian.Uint16(buf[:2])
// 	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
// 	logrus.Debug(fmt.Sprintf("Connecting %s", destAddrPort))
// 	dest, err := net.Dial("udp", destAddrPort)
// 	if err != nil {
//		return nil, destAddrPort, errors.New("dial dst:" + err.Error())
// 	}
// 	logrus.Debug(fmt.Sprintf("Connected %s", destAddrPort))
//
// }

func Socks5Connect(client net.Conn) (net.Conn, string, error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(client, buf[:4])
	if n != 4 {
		return nil, "", errors.New("read header:" + err.Error())
	}
	ver, cmd, _, atyp := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 {
		return nil, "", errors.New("invalid ver")
	}
	if cmd == 3 {
		return nil, "", errors.New("not support UDP")
	}
	if cmd != 1 {
		return nil, "", errors.New("invalid cmd, only support connect")
	}
	addr := ""
	switch atyp {
	case 1:
		n, err = io.ReadFull(client, buf[:4])
		if n != 4 {
			return nil, "", errors.New("invalid IPv4:" + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
	case 3:
		n, err = io.ReadFull(client, buf[:1])
		if n != 1 {
			return nil, "", errors.New("invalid hostname:" + err.Error())
		}
		addrLen := int(buf[0])
		n, err = io.ReadFull(client, buf[:addrLen])
		if n != addrLen {
			return nil, "", errors.New("invalid hostname:" + err.Error())
		}
		addr = string(buf[:addrLen])
	case 4:
		return nil, "", errors.New("IPv6: no supported yet")
	default:
		return nil, "", errors.New("invalid atyp")
	}
	n, err = io.ReadFull(client, buf[:2])
	if n != 2 {
		return nil, "", errors.New("read port:" + err.Error())
	}
	port := binary.BigEndian.Uint16(buf[:2])
	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	logrus.Debug(fmt.Sprintf("Connecting %s", destAddrPort))
	dest, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, destAddrPort, errors.New("dial dst:" + err.Error())
	}
	logrus.Debug(fmt.Sprintf("Connected %s", destAddrPort))
	_, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		dest.Close()
		return nil, destAddrPort, errors.New("write rsp:" + err.Error())
	}
	return dest, destAddrPort, nil
}

/*
func forward(i interface{}) {
	rc := i.(*RelayConn)
	defer rc.src.Close()
	defer rc.dst.Close()
	io.Copy(rc.src, rc.dst)
}

func gforward(i interface{}) {
	rc := i.(*RelayConn)
	defer rc.dst.Close()
	defer rc.src.Close()
	CopyPileline(rc.dst, rc.src, rc.destAddrPort)
}

func Socks5Relay(client, target net.Conn, destAddrPort string) {
	rc := &RelayConn{
		src:          client,
		dst:          target,
		destAddrPort: destAddrPort,
	}
	logrus.Debug(fmt.Sprintf("dpool: %d left: %d, gpool: %d left: %d", dpool.Running(), dpool.Free(), gpool.Running(), gpool.Free()))
	dpool.Invoke(rc)
	if cache.Contains(destAddrPort) {
		logrus.Debug(fmt.Sprintf("Hit LRU Relay Cache: %s", destAddrPort))
		rc.src, rc.dst = rc.dst, rc.src
		dpool.Invoke(rc)
	} else {
		gpool.Invoke(rc)
	}
}
*/

func Socks5Forward(client, target net.Conn, destAddrPort string) {
	forward := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		io.Copy(src, dest)
	}

	gforward := func(dst, src net.Conn) {
		defer dst.Close()
		defer src.Close()
		CopyPileline(dst, src, destAddrPort)
	}

	go forward(client, target)
	if cache.Contains(destAddrPort) {
		logrus.Debug(fmt.Sprintf("Hit LRU Relay Cache: %s", destAddrPort))
		go forward(target, client)
	} else {
		go gforward(target, client)
	}
}

func CopyPileline(dst io.Writer, src io.Reader, destAddrPort string) {
	buf := make([]byte, RDBUF)
	nr, err := src.Read(buf)
	if err != nil {
		if err == io.EOF {
			logrus.Debug(fmt.Sprintf("[%s][%s] read EOF in first phase", destAddrPort, src.(*net.TCPConn).RemoteAddr().String()))
		} else if strings.Contains(err.Error(), "use of closed network connection") {
			logrus.Debug(fmt.Sprintf("[%s][%s] read closed in first phase: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), err.Error()))
		} else {
			logrus.Error(fmt.Sprintf("[%s][%s] read error in first phase: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), err.Error()))
		}
		return
	}
	if nr == 0 {
		logrus.Debug(fmt.Sprintf("[%s][%s] read 0 in first phase", destAddrPort, src.(*net.TCPConn).RemoteAddr().String()))
		return
	}
	hint := string(buf[0:7])
	is_http := false
	for _, v := range HTTP_METHOD {
		if strings.HasPrefix(hint, v) {
			is_http = true
			break
		}
	}
	if !is_http {
		dst.Write(buf[0:nr])
		io.Copy(dst, src)
		cache.Add(destAddrPort, destAddrPort)
		logrus.Debug(fmt.Sprintf("Not HTTP, Hint: %v, Add LRU Relay Cache: %s, Cache Len: %d", buf[0:7], destAddrPort, cache.Len()))
		return
	}
	for {
		parser := http.NewHTTPParser()
		httpBodyOffset, err := parser.Parse(buf[0:nr])
		for err == http.ErrMissingData {
			var m int
			m, err = src.Read(buf[nr:])
			if err != nil {
				logrus.Error(fmt.Sprintf("[%s] read error in http accumulation: %v", destAddrPort, err))
				break
			}
			nr += m
			httpBodyOffset, err = parser.Parse(buf[:nr])
		}
		value, start, end := parser.FindHeader([]byte("User-Agent"))
		if value != nil && end > start {
			if slices.Contains(whitelist, string(value)) {
				logrus.Debug(fmt.Sprintf("[%s][%s] Hit User-Agent Whitelist: %s, Add LRU Relay Cache, Cache Len: %d", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), string(value), cache.Len()))
				dst.Write(buf[0:nr])
				io.Copy(dst, src)
				cache.Add(destAddrPort, destAddrPort)
				return
			}
			logrus.Debug(fmt.Sprintf("[%s][%s] Hit User-Agent: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), string(value)))
			for i := start; i < end; i++ {
				buf[i] = 32
			}
			for i := range payloadByte {
				if start+i >= end {
					break
				}
				buf[start+i] = payloadByte[i]
			}
		} else {
			logrus.Debug(fmt.Sprintf("[%s] Not found User-Agent, Add LRU Relay Cache, Cache Len: %d", destAddrPort, cache.Len()))
			dst.Write(buf[0:nr])
			io.Copy(dst, src)
			cache.Add(destAddrPort, destAddrPort)
			return
		}
		bodyLen := int(parser.ContentLength())
		if bodyLen == -1 {
			bodyLen = 0
		}

		_, ew := dst.Write(buf[0:min(httpBodyOffset+bodyLen, nr)])
		if ew != nil {
			logrus.Error(fmt.Sprintf("[%s][%s] write error: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), ew.Error()))
			break
		}
		if httpBodyOffset+bodyLen > nr {
			left := httpBodyOffset + bodyLen - nr
			for left > 0 {
				lr := min(left, RDBUF)
				m, err := src.Read(buf[0:lr])
				if err != nil {
					logrus.Error(fmt.Sprintf("[%s][%s] read error in large body: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), err.Error()))
					break
				}
				_, ew := dst.Write(buf[0:m])
				if ew != nil {
					logrus.Error(fmt.Sprintf("[%s][%s] write error in large body: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), ew.Error()))
					break
				}
				left -= m
			}
			nr = 0
		} else if httpBodyOffset+bodyLen < nr {
			copy(buf[0:], buf[httpBodyOffset+bodyLen:])
			nr = nr - httpBodyOffset - bodyLen
		} else {
			nr = 0
		}

		m, err := src.Read(buf[nr:])
		nr += m
		if err != nil {
			if err == io.EOF {
				logrus.Debug(fmt.Sprintf("[%s][%s] read EOF in next phase", destAddrPort, src.(*net.TCPConn).RemoteAddr().String()))
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				logrus.Debug(fmt.Sprintf("[%s][%s] read closed in next phase: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), err.Error()))
			} else {
				logrus.Error(fmt.Sprintf("[%s][%s] read error in next phase: %s", destAddrPort, src.(*net.TCPConn).RemoteAddr().String(), err.Error()))
			}
			break
		}
	}
}
