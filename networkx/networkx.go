// Package networkx provides network and IP helpers.
package networkx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var privateCIDRs = mustParseCIDRs([]string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
})

// ClientIP returns the likely client IP from proxy headers and RemoteAddr.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	headerKeys := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Client-IP",
	}
	for _, key := range headerKeys {
		for _, candidate := range splitHeaderIPs(r.Header.Get(key)) {
			ip := NormalizeIP(candidate)
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return NormalizeIP(host)
	}
	return NormalizeIP(r.RemoteAddr)
}

// NormalizeIP trims and validates an IP string.
func NormalizeIP(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "[]")
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}

// IsIP reports whether value is an IPv4 or IPv6 address.
func IsIP(value string) bool {
	return net.ParseIP(strings.TrimSpace(value)) != nil
}

// IsIPv4 reports whether value is an IPv4 address.
func IsIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil
}

// IsIPv6 reports whether value is an IPv6 address.
func IsIPv6(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() == nil
}

// IsPrivateIP reports whether value is loopback, link-local, or private range.
func IsPrivateIP(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return false
	}
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// IsPublicIP reports whether value is a valid non-private IP.
func IsPublicIP(value string) bool {
	return IsIP(value) && !IsPrivateIP(value)
}

// InCIDR reports whether ipValue is contained in cidrValue.
func InCIDR(ipValue, cidrValue string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipValue))
	if ip == nil {
		return false
	}
	_, cidr, err := net.ParseCIDR(strings.TrimSpace(cidrValue))
	if err != nil {
		return false
	}
	return cidr.Contains(ip)
}

// JoinHostPort joins host and port safely, including IPv6 hosts.
func JoinHostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// SplitHostPort splits a host:port string.
func SplitHostPort(address string) (host string, port int, err error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(portText)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

// LocalIPs returns non-loopback local interface IP addresses.
func LocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(addrs))
	seen := map[string]struct{}{}
	for _, addr := range addrs {
		var ip net.IP
		switch typed := addr.(type) {
		case *net.IPNet:
			ip = typed.IP
		case *net.IPAddr:
			ip = typed.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		text := ip.String()
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		result = append(result, text)
	}
	return result, nil
}

// FreePort returns an available TCP port on host. Use host "127.0.0.1" or
// "localhost" for local-only listeners.
func FreePort(host string) (int, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("networkx: listener address is not TCP")
	}
	return addr.Port, nil
}

// TCPReachable reports whether address accepts a TCP connection before timeout.
func TCPReachable(ctx context.Context, address string, timeout time.Duration) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func splitHeaderIPs(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func mustParseCIDRs(values []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, cidr, err := net.ParseCIDR(value)
		if err != nil {
			panic(err)
		}
		result = append(result, cidr)
	}
	return result
}
