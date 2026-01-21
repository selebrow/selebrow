package dialer

import (
	"context"
	"errors"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/mocks"
)

// Test that PerHostDialer uses bypass dialer when proxy is not required.
func TestPerHostDialer_DialContext_BypassProxy(t *testing.T) {
	g := NewWithT(t)

	mockProxy := mocks.NewContextDialer(t)
	mockBypass := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	proxyFunc := func(u *url.URL) (*url.URL, error) {
		if u.Hostname() == "bypass.com" {
			return nil, nil
		}
		return &url.URL{Scheme: "socks5", Host: "localhost:1080"}, nil
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	mockBypass.EXPECT().
		DialContext(t.Context(), "tcp", "bypass.com:443").
		Return(mockConn, nil).
		Once()

	conn, err := dialerInst.DialContext(t.Context(), "tcp", "bypass.com:443")

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(conn).Should(Equal(mockConn))
}

// Test that proxy dialer is used when proxy is required.
func TestPerHostDialer_DialContext_UseProxy(t *testing.T) {
	g := NewWithT(t)

	mockProxy := mocks.NewContextDialer(t)
	mockBypass := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	proxyFunc := func(u *url.URL) (*url.URL, error) {
		if u.Hostname() == "proxy.com" {
			return &url.URL{Scheme: "http", Host: "proxy:3128"}, nil
		}
		return nil, nil
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	mockProxy.EXPECT().
		DialContext(t.Context(), "tcp", "proxy.com:443").
		Return(mockConn, nil).
		Once()

	conn, err := dialerInst.DialContext(t.Context(), "tcp", "proxy.com:443")

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(conn).Should(Equal(mockConn))
}

// Test that error from proxyFunc is propagated.
func TestPerHostDialer_ProxyFuncError(t *testing.T) {
	g := NewWithT(t)

	mockProxy := mocks.NewContextDialer(t)
	mockBypass := mocks.NewContextDialer(t)

	proxyFunc := func(u *url.URL) (*url.URL, error) {
		return nil, errors.New("proxyFunc error")
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	_, err := dialerInst.DialContext(t.Context(), "tcp", "error.com:443")

	g.Expect(err).Should(MatchError("proxyFunc error"))
}

// Test handling of malformed address.
func TestPerHostDialer_InvalidAddress(t *testing.T) {
	g := NewWithT(t)

	mockProxy := mocks.NewContextDialer(t)
	mockBypass := mocks.NewContextDialer(t)

	proxyFunc := func(u *url.URL) (*url.URL, error) {
		return nil, nil
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	_, err := dialerInst.DialContext(t.Context(), "tcp", "invalid::host")
	g.Expect(err).Should(HaveOccurred())
}

// Test that Dial (without context) uses background context.
func TestPerHostDialer_Dial_Shortcut(t *testing.T) {
	g := NewWithT(t)

	mockBypass := mocks.NewContextDialer(t)
	mockProxy := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	proxyFunc := func(u *url.URL) (*url.URL, error) {
		return nil, nil
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	mockBypass.EXPECT().
		DialContext(context.Background(), "tcp", "direct.com:80").
		Return(mockConn, nil).
		Once()

	conn, err := dialerInst.Dial("tcp", "direct.com:80")

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(conn).Should(Equal(mockConn))
}

// Test that getDialer correctly parses address with https:// prefix.
func TestPerHostDialer_GetDialer_ParsesAddressCorrectly(t *testing.T) {
	g := NewWithT(t)

	mockProxy := mocks.NewContextDialer(t)
	mockBypass := mocks.NewContextDialer(t)

	called := false
	proxyFunc := func(u *url.URL) (*url.URL, error) {
		g.Expect(u.Scheme).Should(Equal("https"))
		g.Expect(u.Hostname()).Should(Equal("test.local"))
		g.Expect(u.Port()).Should(Equal("443"))
		called = true
		return nil, nil
	}

	dialerInst := NewPerHostDialer(mockProxy, mockBypass, proxyFunc)

	mockBypass.EXPECT().
		DialContext(t.Context(), "tcp", "test.local:443").
		Return(nil, nil).
		Once()

	_, err := dialerInst.DialContext(t.Context(), "tcp", "test.local:443")
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(called).Should(BeTrue())
}
