package networkx

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	if got := ClientIP(req); got != "127.0.0.1" {
		t.Fatalf("ClientIP remote = %q, want 127.0.0.1", got)
	}

	req.Header.Set("X-Forwarded-For", " 8.8.8.8, 10.0.0.1")
	if got := ClientIP(req); got != "8.8.8.8" {
		t.Fatalf("ClientIP forwarded = %q, want 8.8.8.8", got)
	}
}

func TestIPHelpers(t *testing.T) {
	if !IsIP("8.8.8.8") || !IsIPv4("8.8.8.8") || IsIPv4("2001:4860:4860::8888") {
		t.Fatal("IPv4 helpers returned unexpected result")
	}
	if !IsIPv6("2001:4860:4860::8888") {
		t.Fatal("IsIPv6 should accept IPv6")
	}
	if !IsPrivateIP("192.168.1.1") || IsPrivateIP("8.8.8.8") {
		t.Fatal("IsPrivateIP returned unexpected result")
	}
	if !IsPublicIP("8.8.8.8") || !InCIDR("192.168.1.8", "192.168.1.0/24") {
		t.Fatal("public/cidr helpers returned unexpected result")
	}
}

func TestHostPortAndReachable(t *testing.T) {
	address := JoinHostPort("127.0.0.1", 8080)
	host, port, err := SplitHostPort(address)
	if err != nil {
		t.Fatalf("SplitHostPort failed: %v", err)
	}
	if host != "127.0.0.1" || port != 8080 {
		t.Fatalf("host/port = %s/%d", host, port)
	}

	freePort, err := FreePort("127.0.0.1")
	if err != nil {
		t.Fatalf("FreePort failed: %v", err)
	}
	listener, err := net.Listen("tcp", JoinHostPort("127.0.0.1", freePort))
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()
	if !TCPReachable(context.Background(), listener.Addr().String(), time.Second) {
		t.Fatal("TCPReachable should reach listener")
	}
}
