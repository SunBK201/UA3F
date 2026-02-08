package socks5

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/statistics"
	"golang.org/x/net/proxy"
)

type echoServer struct {
	listener net.Listener
	server   *http.Server
	addr     string
}

func NewEchoServer(t *testing.T) *echoServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create echo server listener: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/echo-ua", func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ua))
	})

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for name, values := range r.Header {
			for _, v := range values {
				_, _ = fmt.Fprintf(w, "%s: %s\n", name, v)
			}
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{Handler: mux}

	es := &echoServer{
		listener: listener,
		server:   server,
		addr:     listener.Addr().String(),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	return es
}

func (es *echoServer) close() {
	_ = es.server.Close()
	_ = es.listener.Close()
}

func (es *echoServer) URL(path string) string {
	return fmt.Sprintf("http://%s%s", es.addr, path)
}

// mockRecorder creates a minimal recorder for testing
func mockRecorder() *statistics.Recorder {
	return &statistics.Recorder{
		RewriteRecordList:     statistics.NewRewriteRecordList("/dev/null"),
		PassThroughRecordList: statistics.NewPassThroughRecordList("/dev/null"),
		ConnectionRecordList:  statistics.NewConnectionRecordList("/dev/null"),
	}
}

func TestServerNew(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        1080,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	server := New(cfg, rw, recorder, nil)

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.Cfg != cfg {
		t.Error("config not set correctly")
	}
	if server.Rewriter != rw {
		t.Error("rewriter not set correctly")
	}
	if server.Recorder != recorder {
		t.Error("recorder not set correctly")
	}
	if server.Cache == nil {
		t.Error("cache not initialized")
	}
}

func TestSocks5ProxyUserAgentRewrite(t *testing.T) {
	// Start echo server
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	tests := []struct {
		name          string
		rewriteMode   config.RewriteMode
		targetUA      string
		originalUA    string
		expectedUA    string
		expectRewrite bool
	}{
		{
			name:          "Global mode rewrites UA",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "UA3F-Test-Agent",
			originalUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedUA:    "UA3F-Test-Agent",
			expectRewrite: true,
		},
		{
			name:          "Global mode rewrites custom UA",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "CustomUA",
			originalUA:    "MyCustomBrowser/1.0",
			expectedUA:    "CustomUA",
			expectRewrite: true,
		},
		{
			name:          "Direct mode passes UA through",
			rewriteMode:   config.RewriteModeDirect,
			targetUA:      "ShouldNotAppear",
			originalUA:    "OriginalUserAgent/1.0",
			expectedUA:    "OriginalUserAgent/1.0",
			expectRewrite: false,
		},
		{
			name:          "Whitelist UA Go-http-client is not rewritten",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "ShouldNotAppear",
			originalUA:    "Go-http-client/1.1",
			expectedUA:    "Go-http-client/1.1",
			expectRewrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ServerMode:  config.ServerModeSocks5,
				BindAddress: "127.0.0.1",
				Port:        0, // Will be assigned dynamically
				LogLevel:    "error",
				RewriteMode: tt.rewriteMode,
				UserAgent:   tt.targetUA,
			}

			recorder := mockRecorder()
			rw, err := rewrite.New(cfg, recorder)
			if err != nil {
				t.Fatalf("failed to create rewriter: %v", err)
			}

			// Find an available port
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("failed to find available port: %v", err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()

			cfg.Port = port
			server := New(cfg, rw, recorder, nil)

			if err := server.Start(); err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer func() { _ = server.Close() }()

			// Wait for server to be ready
			time.Sleep(100 * time.Millisecond)

			// Create SOCKS5 dialer
			dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
			if err != nil {
				t.Fatalf("failed to create SOCKS5 dialer: %v", err)
			}

			// Create HTTP client with SOCKS5 proxy
			client := &http.Client{
				Transport: &http.Transport{
					Dial: dialer.Dial,
				},
				Timeout: 5 * time.Second,
			}

			req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if tt.originalUA != "" {
				req.Header.Set("User-Agent", tt.originalUA)
			} else {
				req.Header.Del("User-Agent")
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to send request through proxy: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			receivedUA := string(body)
			if receivedUA != tt.expectedUA {
				t.Errorf("User-Agent mismatch: got %q, want %q", receivedUA, tt.expectedUA)
			}
		})
	}
}

func TestSocks5ProxyConnect(t *testing.T) {
	// Start echo server
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("failed to create SOCKS5 dialer: %v", err)
	}

	// Test basic TCP connection through SOCKS5
	conn, err := dialer.Dial("tcp", echoSrv.addr)
	if err != nil {
		t.Fatalf("failed to dial through SOCKS5: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send HTTP request manually
	httpReq := "GET / HTTP/1.1\r\nHost: " + echoSrv.addr + "\r\n\r\n"
	_, err = conn.Write([]byte(httpReq))
	if err != nil {
		t.Fatalf("failed to send HTTP request: %v", err)
	}

	// Read response
	buf := make([]byte, 1024)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "200 OK") && !strings.Contains(response, "HTTP/1.1 200") {
		t.Errorf("expected HTTP 200 response, got: %s", response)
	}
}

func TestSocks5ProxyConcurrentRequests(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "ConcurrentTestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("failed to create SOCKS5 dialer: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 10 * time.Second,
	}

	const numRequests = 20
	var wg sync.WaitGroup
	results := make(chan string, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to create request: %v", id, err)
				return
			}
			req.Header.Set("User-Agent", fmt.Sprintf("OriginalUA-%d", id))

			resp, err := client.Do(req)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to send request: %v", id, err)
				return
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to read response: %v", id, err)
				return
			}

			results <- string(body)
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all responses have the rewritten UA
	for result := range results {
		if result != "ConcurrentTestUA" {
			t.Errorf("expected 'ConcurrentTestUA', got %q", result)
		}
	}
}

func TestSocks5ProxyDifferentMethods(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "MethodTestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("failed to create SOCKS5 dialer: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 5 * time.Second,
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var body io.Reader
			if method == "POST" || method == "PUT" || method == "PATCH" {
				body = strings.NewReader("test body")
			}

			req, err := http.NewRequest(method, echoSrv.URL("/echo-ua"), body)
			if err != nil {
				t.Fatalf("failed to create %s request: %v", method, err)
			}
			req.Header.Set("User-Agent", "OriginalUA")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to send %s request: %v", method, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if method == "HEAD" {
				// HEAD request doesn't have a body
				return
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			receivedUA := string(respBody)
			if receivedUA != "MethodTestUA" {
				t.Errorf("%s: expected 'MethodTestUA', got %q", method, receivedUA)
			}
		})
	}
}

func TestSocks5ProxyPartialRewrite(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:              config.ServerModeSocks5,
		BindAddress:             "127.0.0.1",
		Port:                    0,
		LogLevel:                "error",
		RewriteMode:             config.RewriteModeGlobal,
		UserAgent:               "ReplacedPart",
		UserAgentRegex:          `Chrome/\d+`,
		UserAgentPartialReplace: true,
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("failed to create SOCKS5 dialer: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	// Original UA contains "Chrome/120" which should be replaced
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	receivedUA := string(body)
	expectedUA := "Mozilla/5.0 ReplacedPart Safari/537.36"
	if receivedUA != expectedUA {
		t.Errorf("partial rewrite mismatch: got %q, want %q", receivedUA, expectedUA)
	}
}

func TestSocks5Handshake(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Test valid SOCKS5 handshake
	t.Run("ValidHandshake", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer func() { _ = conn.Close() }()

		// Send SOCKS5 greeting: version 5, 1 method, no auth
		_, err = conn.Write([]byte{0x05, 0x01, 0x00})
		if err != nil {
			t.Fatalf("failed to send greeting: %v", err)
		}

		// Read server's method selection
		buf := make([]byte, 2)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			t.Fatalf("failed to read method selection: %v", err)
		}

		// Check response: version 5, no auth method
		if buf[0] != 0x05 || buf[1] != 0x00 {
			t.Errorf("unexpected method selection: got %v, want [5 0]", buf)
		}
	})

	// Test invalid SOCKS version
	t.Run("InvalidVersion", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer func() { _ = conn.Close() }()

		// Send invalid version (SOCKS4)
		_, err = conn.Write([]byte{0x04, 0x01, 0x00})
		if err != nil {
			t.Fatalf("failed to send greeting: %v", err)
		}

		// Read response - server should reject or close
		buf := make([]byte, 2)
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err = io.ReadFull(conn, buf)
		// The connection might be closed or return an error, which is expected
		if err == nil && buf[0] == 0x05 && buf[1] == 0x00 {
			t.Error("server should not accept invalid SOCKS version")
		}
	})
}

func TestSocks5ServerClose(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	_ = conn.Close()

	// Close the server
	if err := server.Close(); err != nil {
		t.Errorf("failed to close server: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify server is no longer accepting connections
	conn, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err == nil {
		_ = conn.Close()
		t.Error("expected connection to fail after server close")
	}
}

func TestSocks5MultipleConnections(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "MultiConnUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Open multiple simultaneous connections
	const numConns = 10
	conns := make([]net.Conn, numConns)

	for i := 0; i < numConns; i++ {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
		if err != nil {
			t.Fatalf("failed to create SOCKS5 dialer %d: %v", i, err)
		}

		conn, err := dialer.Dial("tcp", echoSrv.addr)
		if err != nil {
			t.Fatalf("failed to dial connection %d: %v", i, err)
		}
		conns[i] = conn
	}

	// Close all connections
	for i, conn := range conns {
		if conn != nil {
			if err := conn.Close(); err != nil {
				t.Errorf("failed to close connection %d: %v", i, err)
			}
		}
	}
}

func TestSocks5LargePayload(t *testing.T) {
	// Create a simple TCP echo server for large payloads
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create echo listener: %v", err)
	}
	echoAddr := echoListener.Addr().String()

	go func() {
		for {
			conn, err := echoListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()
	defer func() { _ = echoListener.Close() }()

	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeDirect,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", port), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("failed to create SOCKS5 dialer: %v", err)
	}

	conn, err := dialer.Dial("tcp", echoAddr)
	if err != nil {
		t.Fatalf("failed to dial through SOCKS5: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send large payload
	payload := make([]byte, 1024*1024) // 1MB
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	go func() {
		_, _ = conn.Write(payload)
	}()

	// Read response
	response := make([]byte, len(payload))
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if n != len(payload) {
		t.Errorf("response length mismatch: got %d, want %d", n, len(payload))
	}

	// Verify payload integrity
	for i := 0; i < len(payload); i++ {
		if response[i] != payload[i] {
			t.Errorf("payload mismatch at byte %d: got %d, want %d", i, response[i], payload[i])
			break
		}
	}
}

func TestSocks5UDPAssociate(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Test UDP ASSOCIATE request
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// SOCKS5 handshake
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("failed to send greeting: %v", err)
	}

	buf := make([]byte, 2)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("failed to read method selection: %v", err)
	}

	// Send UDP ASSOCIATE request
	// VER CMD RSV ATYP DST.ADDR DST.PORT
	// 0x05 0x03 0x00 0x01 0x00 0x00 0x00 0x00 0x00 0x00
	udpRequest := []byte{0x05, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err = conn.Write(udpRequest)
	if err != nil {
		t.Fatalf("failed to send UDP ASSOCIATE request: %v", err)
	}

	// Read response
	response := make([]byte, 10)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read UDP ASSOCIATE response: %v", err)
	}

	// Check response
	if n < 4 {
		t.Fatalf("response too short: %d bytes", n)
	}
	if response[0] != 0x05 {
		t.Errorf("unexpected version: got %d, want 5", response[0])
	}
	if response[1] != 0x00 {
		t.Errorf("UDP ASSOCIATE failed with reply code: %d", response[1])
	}
}

func TestSocks5BindCommand(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Test BIND request
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// SOCKS5 handshake
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("failed to send greeting: %v", err)
	}

	buf := make([]byte, 2)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("failed to read method selection: %v", err)
	}

	// Send BIND request
	// VER CMD RSV ATYP DST.ADDR DST.PORT
	// 0x05 0x02 0x00 0x01 0x00 0x00 0x00 0x00 0x00 0x00
	bindRequest := []byte{0x05, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err = conn.Write(bindRequest)
	if err != nil {
		t.Fatalf("failed to send BIND request: %v", err)
	}

	// Read first response (should contain bound address)
	response := make([]byte, 32)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read BIND response: %v", err)
	}

	// Check response
	if n < 4 {
		t.Fatalf("response too short: %d bytes", n)
	}
	if response[0] != 0x05 {
		t.Errorf("unexpected version: got %d, want 5", response[0])
	}
	if response[1] != 0x00 {
		t.Errorf("BIND failed with reply code: %d", response[1])
	}
}
