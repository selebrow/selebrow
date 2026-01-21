package handlers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/selebrow/selebrow/internal/proxy/dialer"
	"github.com/selebrow/selebrow/mocks"
)

const testTarget = "example.com:443"

func TestNewProxyHandler(t *testing.T) {
	g := NewWithT(t)

	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	logger := zaptest.NewLogger(t)

	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	g.Expect(handler).NotTo(BeNil())
	g.Expect(handler.transport).To(Equal(mockTransport))
	g.Expect(handler.connDialer).To(Equal(mockDialer))
	g.Expect(handler.proxyConnect).To(BeTrue())
	g.Expect(handler.logger).NotTo(BeNil())
}

func TestProxyHandler_ServeHTTP_HTTPMethod(t *testing.T) {
	g := NewWithT(t)

	mockTransport := mocks.NewRoundTripper(t)
	logger := zaptest.NewLogger(t)

	handler := NewProxyHandler(mockTransport, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", http.NoBody)
	recorder := httptest.NewRecorder()

	mockTransport.EXPECT().
		RoundTrip(mock.Anything).
		Run(func(req *http.Request) {
			g.Expect(req.URL.String()).To(Equal("http://example.com/path"))
		}).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("OK")),
			Header:     http.Header{},
		}, nil).Once()

	handler.ServeHTTP(recorder, req)

	g.Expect(recorder).Should(HaveHTTPStatus(http.StatusOK))
	g.Expect(recorder).Should(HaveHTTPBody("OK"))
}

func TestProxyHandler_ServeHTTP_CONNECT_HijackNotSupported(t *testing.T) {
	g := NewWithT(t)

	observedLogger, logs := observer.New(zap.WarnLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	handler := NewProxyHandler(nil, nil, logger)

	req := httptest.NewRequest(http.MethodConnect, testTarget, http.NoBody)
	recorder := httptest.NewRecorder() // Hijacker not supported

	handler.ServeHTTP(recorder, req)

	g.Expect(recorder).Should(HaveHTTPStatus(http.StatusInternalServerError))
	g.Expect(recorder).Should(HaveHTTPBody("Internal Server Error\n"))

	accLogs := logs.FilterMessage("request failed")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 500)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("error", Equal("hijacking not supported")))
}

func TestProxyHandler_ServeHTTP_CONNECT_HijackFailed(t *testing.T) {
	g := NewWithT(t)

	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	observedLogger, logs := observer.New(zap.WarnLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	mockHijacker := mocks.NewResponseWriterHijacker(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodConnect, testTarget, http.NoBody)

	mockHijacker.EXPECT().Hijack().Return(nil, nil, errors.New("hijack failed")).Once()
	mockHijacker.EXPECT().Header().RunAndReturn(func() http.Header {
		return recorder.Header()
	}).Once()
	mockHijacker.EXPECT().WriteHeader(mock.Anything).Run(func(code int) {
		recorder.WriteHeader(code)
	}).Once()
	mockHijacker.EXPECT().Write(mock.Anything).RunAndReturn(func(data []byte) (int, error) {
		return recorder.Write(data)
	}).Once()

	handler.ServeHTTP(mockHijacker, req)

	g.Expect(recorder).Should(HaveHTTPStatus(http.StatusInternalServerError))
	g.Expect(recorder).Should(HaveHTTPBody("Internal Server Error\n"))

	accLogs := logs.FilterMessage("request failed")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 500)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("error", Equal("connection hijacking failed")))
}

func TestProxyHandler_ServeHTTP_CONNECT_DialFailed(t *testing.T) {
	g := NewWithT(t)

	// Setup dependencies
	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	observedLogger, logs := observer.New(zap.WarnLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	// Create handler under test
	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	// Create CONNECT request
	req := httptest.NewRequest(http.MethodConnect, testTarget, http.NoBody)
	req.Host = testTarget

	// Mock that combines http.ResponseWriter and http.Hijacker
	mockWriterHijacker := mocks.NewResponseWriterHijacker(t)

	// Expect DialContext to be called and fail
	mockDialer.EXPECT().
		DialContext(
			mock.Anything,
			"tcp",
			testTarget,
		).
		Return(nil, errors.New("dial failed")).
		Once()

	// Since the hijack is successful, we expect Hijack() to be called
	// Because dialing to target fails, rawHttpError writes directly to hijacked connection
	// We expect Write() and Close() on the hijacked connection
	hijackedConn := mocks.NewConn(t)
	mockWriterHijacker.EXPECT().Hijack().
		Return(hijackedConn, nil, nil).
		Once()

	// Expect raw HTTP/1.1 error response written via hijacked connection
	hijackedConn.EXPECT().
		Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Length: 11\r\nContent-Type: text/plain; charset=utf-8\r\n\r\ndial failed")).
		RunAndReturn(func(data []byte) (int, error) {
			return len(data), nil
		}).
		Once()

	// The hijacked connection should be closed after error
	hijackedConn.EXPECT().Close().Return(nil).Once()

	// Act: Serve the CONNECT request
	handler.ServeHTTP(mockWriterHijacker, req)

	// Assert: logs
	accLogs := logs.FilterMessage("request failed")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 502)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("error", Equal("dial failed")))
}

// where the proxy establishes a tunnel between client and target server.
func TestProxyHandler_ServeHTTP_CONNECT_Success(t *testing.T) {
	g := NewWithT(t)

	// Setup dependencies
	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	observedLogger, logs := observer.New(zap.DebugLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	// Create the handler under test
	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	// Create CONNECT request
	expectedReq := httptest.NewRequest(http.MethodConnect, testTarget, http.NoBody)

	// Mock that combines http.ResponseWriter and http.Hijacker
	mockWriterHijacker := mocks.NewResponseWriterHijacker(t)

	// Mock the hijacked client connection (between client and proxy)
	clientConn := mocks.NewConn(t)

	// Mock the target connection (between proxy and target server)
	targetConn := mocks.NewConn(t)

	// Expect Hijack() to be called once and return the client connection
	mockWriterHijacker.EXPECT().Hijack().
		Return(clientConn, nil, nil).
		Once()

	// Expect DialContext to be called with the correct arguments and return target connection
	mockDialer.EXPECT().
		DialContext(mock.Anything, "tcp", testTarget).
		RunAndReturn(func(ctx context.Context, _ string, _ string) (net.Conn, error) {
			maybeReq := ctx.Value(dialer.ProxyRequestKey)
			actualReq, ok := maybeReq.(*http.Request)
			g.Expect(ok).To(BeTrue())
			g.Expect(actualReq.URL).To(Equal(expectedReq.URL))
			g.Expect(actualReq.Method).To(Equal(expectedReq.Method))
			g.Expect(actualReq.Header).To(Equal(http.Header{"User-Agent": []string{""}}))
			return targetConn, nil
		}).
		Once()

	// Expect the proxy to send "200 Connection Established" over the hijacked connection
	clientConn.EXPECT().Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")).
		RunAndReturn(func(data []byte) (int, error) {
			return len(data), nil
		}).
		Once()

	// === Simulate client -> target flow ===
	// Client sends data
	clientConn.EXPECT().Read(mock.Anything).RunAndReturn(
		func(p []byte) (int, error) {
			data := []byte("Client to target\n")
			return copy(p, data), nil
		},
	).Once()

	// Proxy writes it to target
	targetConn.EXPECT().Write([]byte("Client to target\n")).
		RunAndReturn(func(data []byte) (int, error) {
			return len(data), nil
		}).Once()

	// === Simulate target -> client flow ===
	// Target sends response
	targetConn.EXPECT().Read(mock.Anything).RunAndReturn(
		func(p []byte) (int, error) {
			data := []byte("Target to client\n")
			return copy(p, data), nil
		},
	).Once()

	// Proxy writes it back to client
	clientConn.EXPECT().Write([]byte("Target to client\n")).
		RunAndReturn(func(data []byte) (int, error) {
			return len(data), nil
		}).Once()

	// Allow graceful shutdown: next Read returns EOF
	clientConn.EXPECT().Read(mock.Anything).Return(
		0, io.EOF,
	).Once()

	targetConn.EXPECT().Read(mock.Anything).Return(
		0, io.EOF,
	).Once()

	// Expect both connections to be closed
	clientConn.EXPECT().Close().Return(nil).Once()
	targetConn.EXPECT().Close().Return(nil).Once()

	// Act: Serve the CONNECT request
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(mockWriterHijacker, expectedReq)
	}()

	// Wait for the handler to finish
	g.Eventually(done).Should(BeClosed(), "Handler did not finish within timeout")

	// Assert: logs
	accLogs := logs.FilterMessage("connection established")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 200)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("host", Equal(testTarget)))

	accLogs = logs.FilterMessage("connection closed")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 200)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("host", Equal(testTarget)))
}

// TestProxyHandler_ServeHTTP_CONNECT_CopyWithError_LogsAndCloses verifies that
// when an error occurs during data copying in the tunnel (e.g., broken connection),
// the error is logged, and both connections are closed properly.
func TestProxyHandler_ServeHTTP_CONNECT_CopyWithError_LogsAndCloses(t *testing.T) {
	g := NewWithT(t)

	// Setup dependencies
	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	observedLogger, logs := observer.New(zap.WarnLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	// Create CONNECT request
	req := httptest.NewRequest(http.MethodConnect, testTarget, http.NoBody)

	// Mock ResponseWriter + Hijacker
	mockWriterHijacker := mocks.NewResponseWriterHijacker(t)

	// Mock connections
	clientConn := mocks.NewConn(t)
	targetConn := mocks.NewConn(t)

	// Expect Hijack() to succeed
	mockWriterHijacker.EXPECT().Hijack().
		Return(clientConn, nil, nil).
		Once()

	// Expect successful dial to target
	mockDialer.EXPECT().
		DialContext(mock.Anything, "tcp", testTarget).
		Return(targetConn, nil).
		Once()

	// Expect "200 Connection Established" sent to client
	clientConn.EXPECT().Write(mock.Anything).
		RunAndReturn(func(data []byte) (int, error) {
			g.Expect(string(data)).Should(ContainSubstring("200 Connection Established"))
			return len(data), nil
		}).
		Once()

	// === Simulate: client -> target flow fails on Write ===
	// Client sends data
	clientConn.EXPECT().Read(mock.Anything).RunAndReturn(
		func(p []byte) (int, error) {
			data := []byte("Client to target\n")
			return copy(p, data), nil
		},
	).Once()

	// Proxy tries to write to target, but Write fails
	writeErr := errors.New("write to target failed")
	targetConn.EXPECT().Write([]byte("Client to target\n")).
		Return(0, writeErr).
		Once()

	targetConn.EXPECT().Read(mock.Anything).Return(
		0, io.EOF,
	).Once()

	// After error, both connections should be closed
	targetConn.EXPECT().Close().Return(nil).Once()
	clientConn.EXPECT().Close().Return(nil).Once()

	// Act: Serve the CONNECT request
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(mockWriterHijacker, req)
	}()

	// Wait for the handler to finish
	g.Eventually(done).Should(BeClosed(), "Handler did not finish after i/o error")

	// Assert: logs
	accLogs := logs.FilterMessage("error copying connection data")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].Level).To(Equal(zapcore.WarnLevel))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("host", Equal(testTarget)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("error", Equal(writeErr.Error())))
}

func TestProxyHandler_handleHTTP_ModifyResponseLogs(t *testing.T) {
	g := NewWithT(t)

	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	observedLogger, logs := observer.New(zap.DebugLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
	))

	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", http.NoBody)
	recorder := httptest.NewRecorder()

	mockTransport.EXPECT().
		RoundTrip(mock.Anything).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("Not Found")),
			Header:     http.Header{},
		}, nil).Once()

	handler.ServeHTTP(recorder, req)

	g.Expect(recorder).Should(HaveHTTPStatus(http.StatusNotFound))
	g.Expect(recorder).Should(HaveHTTPBody("Not Found"))

	accLogs := logs.FilterMessage("request completed")
	g.Expect(accLogs.All()).Should(HaveLen(1))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("status", BeNumerically("==", 404)))
	g.Expect(accLogs.All()[0].ContextMap()).To(HaveKeyWithValue("latency", BeNumerically(">", 0)))
}

func TestProxyHandler_handleHTTP_RoundTripFails_ErrorHandlerCalled(t *testing.T) {
	g := NewWithT(t)

	mockTransport := mocks.NewRoundTripper(t)
	mockDialer := mocks.NewContextDialer(t)
	logger := zaptest.NewLogger(t)

	handler := NewProxyHandler(mockTransport, mockDialer, logger)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/fail", http.NoBody)
	recorder := httptest.NewRecorder()

	expectedErr := errors.New("failed to reach upstream")
	mockTransport.EXPECT().
		RoundTrip(mock.Anything).
		Return(nil, expectedErr).
		Once()

	handler.ServeHTTP(recorder, req)

	g.Expect(recorder).Should(HaveHTTPStatus(http.StatusBadGateway))
	g.Expect(recorder).Should(HaveHTTPBody("Bad Gateway\n"))
}
