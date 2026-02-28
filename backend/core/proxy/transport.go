package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	xproxy "golang.org/x/net/proxy"
)

func ApplyProxyToTransport(transport *http.Transport, proxyURL *url.URL, timeout time.Duration) error {
	if transport == nil {
		return fmt.Errorf("transport is nil")
	}

	dialTimeout := timeout
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}

	baseDialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	transport.Proxy = nil
	transport.DialContext = baseDialer.DialContext

	if proxyURL == nil {
		return nil
	}

	scheme := strings.ToLower(strings.TrimSpace(proxyURL.Scheme))
	if scheme == "" {
		scheme = "http"
	}

	switch scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
		return nil
	case "socks5", "socks5h":
		dialer, err := xproxy.FromURL(proxyURL, baseDialer)
		if err != nil {
			return err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			type dialResult struct {
				conn net.Conn
				err  error
			}
			resultChan := make(chan dialResult, 1)
			go func() {
				conn, dialErr := dialer.Dial(network, addr)
				resultChan <- dialResult{conn: conn, err: dialErr}
			}()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case result := <-resultChan:
				return result.conn, result.err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
}
