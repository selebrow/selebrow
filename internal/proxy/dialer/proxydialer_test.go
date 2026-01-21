package dialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/selebrow/selebrow/mocks"
)

const testProxyAddr = "example-proxy:8080"

var (
	proxyURL, _ = url.Parse("http://" + testProxyAddr)
	testReq     = &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: "example.com:443"},
		Host:   "example.com",
		Header: http.Header{"User-Agent": []string{}},
	}
)

func TestProxyDialer_DialContext_MissingProxyRequestInContext(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	dialer := NewProxyDialer(proxyURL, mockDialer)

	// No proxy request in context
	conn, err := dialer.DialContext(t.Context(), "tcp", "example.com:443")

	g.Expect(conn).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError("proxy request is not set in the context"))
}

func TestProxyDialer_DialContext_FailToConnectToProxy(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	dialer := NewProxyDialer(proxyURL, mockDialer)

	ctx := context.WithValue(t.Context(), ProxyRequestKey, testReq)

	// Expect dial failure
	mockDialer.EXPECT().
		DialContext(ctx, "tcp", testProxyAddr).
		Return(nil, errors.New("connection refused")).
		Once()

	conn, err := dialer.DialContext(ctx, "tcp", "example.com:443")

	g.Expect(conn).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(MatchRegexp(".*connection refused")))
}

func TestProxyDialer_DialContext_FailToSendRequest(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	dialer := NewProxyDialer(proxyURL, mockDialer)

	ctx := context.WithValue(t.Context(), ProxyRequestKey, testReq)

	mockDialer.EXPECT().
		DialContext(ctx, "tcp", testProxyAddr).
		Return(mockConn, nil).
		Once()

	// Simulate write failure
	mockConn.EXPECT().Write([]byte("CONNECT example.com HTTP/1.1\r\nHost: example.com\r\n\r\n")).
		Run(func(b []byte) {
			fmt.Println(string(b))
		}).
		Return(0, errors.New("write failed")).
		Once()
	mockConn.EXPECT().Close().Return(nil).Once()

	conn, err := dialer.DialContext(ctx, "tcp", "example.com:443")

	g.Expect(conn).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(MatchRegexp(".*write failed")))
}

func TestProxyDialer_DialContext_FailToReadResponse(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	dialer := NewProxyDialer(proxyURL, mockDialer)

	ctx := context.WithValue(t.Context(), ProxyRequestKey, testReq)

	mockDialer.EXPECT().
		DialContext(ctx, "tcp", testProxyAddr).
		Return(mockConn, nil).
		Once()

	// Mock Write success
	mockConn.EXPECT().Write([]byte("CONNECT example.com HTTP/1.1\r\nHost: example.com\r\n\r\n")).
		RunAndReturn(func(p []byte) (int, error) {
			return len(p), nil
		}).
		Once()

	// Mock Read error via bufio.Reader → http.ReadResponse
	mockConn.EXPECT().Read(mock.Anything).
		Return(0, errors.New("read error")).
		Once()
	mockConn.EXPECT().Close().Return(nil).Once()

	conn, err := dialer.DialContext(ctx, "tcp", "example.com:443")

	g.Expect(conn).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(MatchRegexp(".*read error")))
}

func TestProxyDialer_DialContext_NonSuccessStatus(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	dialer := NewProxyDialer(proxyURL, mockDialer)

	ctx := context.WithValue(t.Context(), ProxyRequestKey, testReq)

	mockDialer.EXPECT().
		DialContext(ctx, "tcp", testProxyAddr).
		Return(mockConn, nil).
		Once()

	mockConn.EXPECT().Write([]byte("CONNECT example.com HTTP/1.1\r\nHost: example.com\r\n\r\n")).
		RunAndReturn(func(p []byte) (int, error) {
			return len(p), nil
		}).
		Once()

	// Return 407 response
	mockConn.EXPECT().Read(mock.Anything).
		RunAndReturn(func(p []byte) (int, error) {
			resp := "HTTP/1.1 407 Proxy Auth Required\r\n\r\n"
			return copy(p, resp), io.EOF
		}).
		Once()

	mockConn.EXPECT().Close().Return(nil).Once()

	conn, err := dialer.DialContext(ctx, "tcp", "example.com:443")

	g.Expect(conn).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(MatchRegexp(".*407 Proxy Auth Required")))
}

func TestProxyDialer_DialContext_Success(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	mockConn := mocks.NewConn(t)

	dialer := NewProxyDialer(proxyURL, mockDialer)

	ctx := context.WithValue(t.Context(), ProxyRequestKey, testReq)

	mockDialer.EXPECT().
		DialContext(ctx, "tcp", testProxyAddr).
		Return(mockConn, nil).
		Once()

	mockConn.EXPECT().Write([]byte("CONNECT example.com HTTP/1.1\r\nHost: example.com\r\n\r\n")).
		RunAndReturn(func(p []byte) (int, error) {
			return len(p), nil
		}).
		Once()

	// Simulate reading "HTTP/1.1 200 OK"
	mockConn.EXPECT().Read(mock.Anything).
		RunAndReturn(func(p []byte) (int, error) {
			const response = "HTTP/1.1 200 OK\r\n\r\n"
			n := copy(p, response)
			return n, io.EOF // EOF is okay — http.ReadResponse handles it after headers
		}).
		Once()

	conn, err := dialer.DialContext(ctx, "tcp", "example.com:443")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(conn).To(Equal(mockConn))
}

func TestProxyDialer_Dial(t *testing.T) {
	g := NewWithT(t)

	mockDialer := mocks.NewContextDialer(t)
	dialer := NewProxyDialer(proxyURL, mockDialer)

	// Dial uses background context — so we expect failure due to missing proxy request
	_, err := dialer.Dial("tcp", "example.com:443")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError("proxy request is not set in the context"))
}
