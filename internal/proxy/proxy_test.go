package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestProxy_Start_CleanShutdown(t *testing.T) {
	g := NewWithT(t)

	logger := zaptest.NewLogger(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	listen, err := listenFreePort()
	g.Expect(err).NotTo(HaveOccurred())

	proxy := NewProxy(handler, listen, logger)
	serverURL := "http://" + listen

	done := make(chan struct{})
	go func() {
		defer close(done)
		proxy.Start()
	}()

	// Wait for server to be ready
	g.Eventually(func() error {
		resp, err := http.Get(serverURL + "/test")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		return err
	}).Should(Succeed())

	// Trigger shutdown
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()
	err = proxy.Shutdown(ctx)
	g.Expect(err).NotTo(HaveOccurred())

	// Expect clean return from Start()
	g.Eventually(done).Should(BeClosed())
}

func TestProxy_Start_FatalOnRealError(t *testing.T) {
	g := NewWithT(t)

	// Create an observer to capture logs
	observedLogger, logs := observer.New(zap.FatalLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, observedLogger)
		}),
		zap.WithFatalHook(zapcore.WriteThenPanic), // Prevent os.Exit, Panic instead
	))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	proxy := NewProxy(handler, ":-1", logger) // Invalid port → bind error

	// Capture when Start() returns
	done := make(chan struct{})
	recovered := false
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		proxy.Start()
	}()

	// Expect Start() to return (after "fatal" log, converted by hook)
	g.Eventually(done).Should(BeClosed(), "proxy.Start() should return after fatal-level log")
	g.Expect(recovered).To(BeTrue(), "Start() should panic after fatal log")

	// Check that there's a fatal log with expected message
	errLogs := logs.FilterMessage("failed to start proxy")
	g.Expect(errLogs.Len()).To(Equal(1), "expected exactly one fatal log entry")
	g.Expect(errLogs.All()[0].Level).To(Equal(zap.FatalLevel), "expected fatal log level")
}

func listenFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return fmt.Sprintf("localhost:%d", l.Addr().(*net.TCPAddr).Port), nil
}
