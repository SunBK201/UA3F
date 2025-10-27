package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/statistics"
)

// SOCKS5 constants
const (
	socksVer5    = 0x05
	socksNoAuth  = 0x00
	socksCmdConn = 0x01
	socksCmdUDP  = 0x03

	socksATYPv4    = 0x01
	socksATYDomain = 0x03
	socksATYPv6    = 0x04
)

var (
	ErrInvalidSocksVersion = errors.New("invalid socks version")
	ErrInvalidSocksCmd     = errors.New("invalid socks cmd")
)

// Server is a minimal SOCKS5 server that delegates HTTP UA rewriting to Rewriter.
type Server struct {
	cfg        *config.Config
	rw         *rewrite.Rewriter
	listener   net.Listener
	ListenAddr string
}

// New returns a new Server with given config, rewriter, and version string.
func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	return &Server{
		cfg:        cfg,
		rw:         rw,
		ListenAddr: fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port),
	}
}

// Start begins listening for SOCKS5 clients.
func (s *Server) Start() (err error) {
	if s.listener, err = net.Listen("tcp", s.ListenAddr); err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	// Start statistics worker
	go statistics.StartRecorder()

	var client net.Conn
	for {
		if client, err = s.listener.Accept(); err != nil {
			logrus.Error("Accept failed: ", err)
			continue
		}
		logrus.Debugf("Accept connection from %s", client.RemoteAddr().String())
		go s.handleClient(client)
	}
}

// handleClient performs SOCKS5 negotiation and dispatches TCP/UDP handling.
func (s *Server) handleClient(client net.Conn) {
	// Handshake (no auth)
	if err := s.socks5Auth(client); err != nil {
		_ = client.Close()
		return
	}

	destAddrPort, cmd, err := s.parseSocks5Request(client)
	if err != nil {
		if cmd == socksCmdUDP {
			// UDP Associate
			s.handleUDPAssociate(client)
			_ = client.Close()
			return
		}
		logrus.Debugf("[%s][%s] ParseSocks5Request failed: %s",
			client.RemoteAddr().String(), destAddrPort, err.Error())
		_ = client.Close()
		return
	}

	// TCP CONNECT
	target, err := s.socks5Connect(client, destAddrPort)
	if err != nil {
		logrus.Debug("Connect failed: ", err)
		_ = client.Close()
		return
	}
	s.forwardTCP(client, target, destAddrPort)
}

// socks5Auth performs a minimal "no-auth" negotiation.
func (s *Server) socks5Auth(client net.Conn) error {
	buf := make([]byte, 256)

	// Read VER, NMETHODS
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		if errors.Is(err, io.EOF) {
			logrus.Warnf("[%s][Auth] read EOF", client.RemoteAddr().String())
		} else {
			logrus.Errorf("[%s][Auth] read header: %v", client.RemoteAddr().String(), err)
		}
		return fmt.Errorf("reading header: %w", err)
	}
	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != socksVer5 {
		logrus.Errorf("[%s][Auth] invalid ver", client.RemoteAddr().String())
		return ErrInvalidSocksVersion
	}

	// Read METHODS
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		logrus.Errorf("[%s][Auth] read methods: %v", client.RemoteAddr().String(), err)
		return fmt.Errorf("read methods: %w", err)
	}

	// Reply: no-auth
	n, err = client.Write([]byte{socksVer5, socksNoAuth})
	if n != 2 || err != nil {
		logrus.Errorf("[%s][Auth] write rsp: %v", client.RemoteAddr().String(), err)
		return fmt.Errorf("write rsp: %w", err)
	}
	return nil
}

// parseSocks5Request reads a single SOCKS5 request. Returns dest, cmd, and error.
func (s *Server) parseSocks5Request(client net.Conn) (string, byte, error) {
	buf := make([]byte, 256)

	// VER, CMD, RSV, ATYP
	if _, err := io.ReadFull(client, buf[:4]); err != nil {
		return "", 0, fmt.Errorf("read header: %w", err)
	}
	ver, cmd, atyp := buf[0], buf[1], buf[3]
	if ver != socksVer5 {
		return "", cmd, ErrInvalidSocksVersion
	}

	// UDP associate: let caller handle
	if cmd == socksCmdUDP {
		return "", socksCmdUDP, errors.New("UDP Associate")
	}
	if cmd != socksCmdConn {
		return "", cmd, ErrInvalidSocksCmd
	}

	var addr string
	switch atyp {
	case socksATYPv4:
		if _, err := io.ReadFull(client, buf[:4]); err != nil {
			return "", cmd, fmt.Errorf("invalid IPv4: %w", err)
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])

	case socksATYDomain:
		if _, err := io.ReadFull(client, buf[:1]); err != nil {
			return "", cmd, fmt.Errorf("invalid hostname(len): %w", err)
		}
		addrLen := int(buf[0])
		if _, err := io.ReadFull(client, buf[:addrLen]); err != nil {
			return "", cmd, fmt.Errorf("invalid hostname: %w", err)
		}
		addr = string(buf[:addrLen])

	case socksATYPv6:
		return "", cmd, errors.New("IPv6: not supported yet")

	default:
		return "", cmd, errors.New("invalid atyp")
	}

	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return "", cmd, fmt.Errorf("read port: %w", err)
	}
	port := binary.BigEndian.Uint16(buf[:2])

	return fmt.Sprintf("%s:%d", addr, port), cmd, nil
}

// socks5Connect dials the target and responds success to the client.
func (s *Server) socks5Connect(client net.Conn, destAddrPort string) (net.Conn, error) {
	logrus.Debugf("Connecting %s", destAddrPort)
	target, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Connected %s", destAddrPort)

	// Reply success (bind set to 0.0.0.0:0)
	if _, err = client.Write([]byte{socksVer5, 0x00, 0x00, socksATYPv4, 0, 0, 0, 0, 0, 0}); err != nil {
		_ = target.Close()
		return nil, err
	}
	return target, nil
}

// forwardTCP proxies traffic in both directions.
// target->client uses raw copy.
// client->target is processed by the rewriter (or raw if cached).
func (s *Server) forwardTCP(client, target net.Conn, destAddrPort string) {
	// Server -> Client (raw)
	go s.copyHalf(client, target)

	// Client -> Server (rewriter)
	go s.proxyHalf(target, client, destAddrPort)
}

// copyHalf copies from src to dst and half-closes both sides when done.
func (s *Server) copyHalf(dst, src net.Conn) {
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
	}()
	_, _ = io.Copy(dst, src)
}

// proxyHalf runs the rewriter proxy on src->dst and then half-closes both sides.
func (s *Server) proxyHalf(dst, src net.Conn, destAddrPort string) {
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
	}()
	_ = s.rw.ProxyHTTPOrRaw(dst, src, destAddrPort)
}

// handleUDPAssociate handles a UDP ASSOCIATE request by creating a UDP relay socket.
// Only IPv4 and domain ATYP are supported (no IPv6).
func (s *Server) handleUDPAssociate(client net.Conn) {
	udpServer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		logrus.Errorf("[%s][UDP] ListenUDP failed: %v", client.RemoteAddr().String(), err)
		return
	}
	defer udpServer.Close()

	_, portStr, _ := net.SplitHostPort(udpServer.LocalAddr().String())
	logrus.Debugf("[%s][UDP] ListenUDP on %s", client.RemoteAddr().String(), portStr)

	portInt, _ := net.LookupPort("udp", portStr)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(portInt))

	// Reply with chosen UDP port (bind addr set to 0.0.0.0)
	if _, err = client.Write([]byte{socksVer5, 0x00, 0x00, socksATYPv4, 0, 0, 0, 0, portBytes[0], portBytes[1]}); err != nil {
		logrus.Errorf("[%s][UDP] Write rsp failed: %v", client.RemoteAddr().String(), err)
		return
	}

	buf := make([]byte, 65535)
	udpPortMap := make(map[string][]byte)
	var clientAddr *net.UDPAddr
	isDomain := false

	for {
		_ = udpServer.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, fromAddr, err := udpServer.ReadFromUDP(buf)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				logrus.Debugf("[%s][UDP] ReadFromUDP timeout: %v", client.RemoteAddr().String(), err)
				if !isAlive(client) {
					logrus.Debugf("[%s][UDP] client is not alive", client.RemoteAddr().String())
					break
				}
			} else {
				logrus.Errorf("[%s][UDP] ReadFromUDP failed: %v", client.RemoteAddr().String(), err)
			}
			continue
		}
		if clientAddr == nil {
			clientAddr = fromAddr
		}

		if clientAddr.IP.Equal(fromAddr.IP) && clientAddr.Port == fromAddr.Port {
			// Packet from client -> forward to remote
			atyp := buf[3]
			var (
				targetAddr string
				targetPort uint16
				payload    []byte
				header     []byte
				targetIP   net.IP
			)

			switch atyp {
			case socksATYPv4:
				isDomain = false
				targetAddr = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
				targetIP = net.ParseIP(targetAddr)
				targetPort = binary.BigEndian.Uint16(buf[8:10])
				payload = buf[10:n]
				header = buf[0:10]

			case socksATYDomain:
				isDomain = true
				addrLen := int(buf[4])
				targetAddr = string(buf[5 : 5+addrLen])
				targetIPAddr, err := net.ResolveIPAddr("ip", targetAddr)
				if err != nil {
					logrus.Errorf("[%s][UDP] ResolveIPAddr failed: %v", client.RemoteAddr().String(), err)
					break
				}
				targetIP = targetIPAddr.IP
				targetPort = binary.BigEndian.Uint16(buf[5+addrLen : 5+addrLen+2])
				payload = buf[5+addrLen+2 : n]
				header = buf[0 : 5+addrLen+2]

			case socksATYPv6:
				logrus.Errorf("[%s][UDP] IPv6: not supported yet", client.RemoteAddr().String())
				return

			default:
				logrus.Errorf("[%s][UDP] invalid atyp", client.RemoteAddr().String())
				return
			}

			remoteAddr := &net.UDPAddr{IP: targetIP, Port: int(targetPort)}
			udpPortMap[remoteAddr.String()] = make([]byte, len(header))
			copy(udpPortMap[remoteAddr.String()], header)

			_ = udpServer.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err = udpServer.WriteToUDP(payload, remoteAddr); err != nil {
				logrus.Debugf("[%s][UDP] WriteToUDP to remote failed: %v", client.RemoteAddr().String(), err)
				continue
			}
		} else {
			// Packet from remote -> forward to client (rebuild header)
			header := udpPortMap[fromAddr.String()]
			if header == nil {
				logrus.Errorf("[%s][UDP] udpPortMap invalid header", client.RemoteAddr().String())
				continue
			}
			// For domain ATYP, preserve original head section size
			if isDomain {
				header = header[0:4]
			}
			body := append(header, buf[:n]...)
			if _, err = udpServer.WriteToUDP(body, clientAddr); err != nil {
				logrus.Debugf("[%s][UDP] WriteToUDP to client failed: %v", client.RemoteAddr().String(), err)
				continue
			}
		}
	}
}

// isAlive checks if a connection is still alive using a short read deadline.
func isAlive(conn net.Conn) bool {
	one := make([]byte, 1)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Read(one)
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			logrus.Debugf("[%s] isAlive: EOF", conn.RemoteAddr().String())
			return false
		case strings.Contains(err.Error(), "use of closed network connection"):
			logrus.Debugf("[%s] isAlive: closed", conn.RemoteAddr().String())
			return false
		case strings.Contains(err.Error(), "i/o timeout"):
			logrus.Debugf("[%s] isAlive: timeout", conn.RemoteAddr().String())
			return true
		default:
			logrus.Debugf("[%s] isAlive: %s", conn.RemoteAddr().String(), err.Error())
			return false
		}
	}
	_ = conn.SetReadDeadline(time.Time{})
	return true
}
