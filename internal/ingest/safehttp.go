package ingest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NewSafeHTTPClient returns a crawler client that cannot connect to loopback,
// private, link-local or otherwise non-public addresses. The guard lives in
// DialContext so redirects and DNS resolution are checked at the actual
// connection boundary, not merely when an administrator enters a feed URL.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream address: %w", err)
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range ips {
			if !publicIP(ip) {
				lastErr = fmt.Errorf("refusing non-public upstream address %s", ip)
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("upstream %q resolved to no addresses", host)
		}
		return nil, lastErr
	}
	return &http.Client{Timeout: timeout, Transport: safeTransport{base: transport}}
}

type safeTransport struct {
	base *http.Transport
}

// RoundTrip validates the destination before proxy selection. Without this
// layer, HTTP_PROXY could make DialContext see only the proxy address while the
// proxy itself was asked to fetch a private hostname.
func (t safeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := ValidatePublicHTTPURL(req.URL.String()); err != nil {
		return nil, err
	}
	host := req.URL.Hostname()
	if net.ParseIP(host) == nil {
		ips, err := net.DefaultResolver.LookupIP(req.Context(), "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("upstream %q resolved to no addresses", host)
		}
		for _, ip := range ips {
			if !publicIP(ip) {
				return nil, fmt.Errorf("refusing non-public upstream address %s", ip)
			}
		}
	}
	return t.base.RoundTrip(req)
}

func publicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

// ValidatePublicHTTPURL performs the cheap, deterministic part of source URL
// validation. DNS is intentionally enforced again by NewSafeHTTPClient.
func ValidatePublicHTTPURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("URL must be absolute http or https")
	}
	if u.User != nil {
		return fmt.Errorf("URL credentials are not allowed")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("localhost is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && !publicIP(ip) {
		return fmt.Errorf("non-public address is not allowed")
	}
	return nil
}
