package proxy

import (
	"context"
	"net"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/mocks"
)

func TestResolveProxyFunc(t *testing.T) {
	g := NewWithT(t)
	targetURL, _ := url.Parse("http://example.com:8080/path")

	t.Run("domain resolves to IP, proxy uses modified URL with IP", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockResolver := mocks.NewHostResolver(t)

		mockResolver.EXPECT().LookupHost(context.Background(), "example.com").
			Return([]string{"1.1.1.1"}, nil)

		called := false
		proxyFunc := func(u *url.URL) (*url.URL, error) {
			called = true
			// Use g.Expect() — properly scoped
			g.Expect(u.Host).To(Equal("1.1.1.1:8080"))
			return &url.URL{Host: "proxy.local:3128"}, nil
		}

		wrapped := ResolveProxyFunc(mockResolver, proxyFunc, logger)
		proxy, err := wrapped(targetURL)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(proxy).ShouldNot(BeNil())
		g.Expect(proxy.Host).To(Equal("proxy.local:3128"))
		g.Expect(called).To(BeTrue())
	})

	t.Run("host is IP, proxyFunc called directly without DNS resolve", func(t *testing.T) {
		logger := zaptest.NewLogger(t)

		ipURL, _ := url.Parse("http://9.9.9.9:1234/path")

		called := false
		proxyFunc := func(u *url.URL) (*url.URL, error) {
			called = true
			g.Expect(u.Host).To(Equal("9.9.9.9:1234"))
			return nil, nil
		}

		wrapped := ResolveProxyFunc(nil, proxyFunc, logger)
		proxy, err := wrapped(ipURL)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(proxy).Should(BeNil())
		g.Expect(called).To(BeTrue())
	})

	t.Run("DNS lookup fails, fallback to original URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockResolver := mocks.NewHostResolver(t)

		mockResolver.EXPECT().LookupHost(context.Background(), "example.com").
			Return([]string{}, &net.DNSError{Err: "server timeout", Name: "example.com"})

		called := false
		proxyFunc := func(u *url.URL) (*url.URL, error) {
			called = true
			g.Expect(u.Host).To(Equal("example.com:8080"))
			return &url.URL{Host: "backup-proxy:3128"}, nil
		}

		wrapped := ResolveProxyFunc(mockResolver, proxyFunc, logger)
		proxy, err := wrapped(targetURL)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(proxy).ShouldNot(BeNil())
		g.Expect(proxy.Host).To(Equal("backup-proxy:3128"))
		g.Expect(called).To(BeTrue())
	})

	t.Run("DNS returns no addresses, fallback to original URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockResolver := mocks.NewHostResolver(t)

		mockResolver.EXPECT().LookupHost(context.Background(), "example.com").
			Return([]string{}, nil)

		called := false
		proxyFunc := func(u *url.URL) (*url.URL, error) {
			called = true
			g.Expect(u.Host).To(Equal("example.com:8080"))
			return &url.URL{Host: "proxy.local:3128"}, nil
		}

		wrapped := ResolveProxyFunc(mockResolver, proxyFunc, logger)
		proxy, err := wrapped(targetURL)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(proxy).ShouldNot(BeNil())
		g.Expect(called).To(BeTrue())
	})
}
