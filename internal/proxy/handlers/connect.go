package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/internal/proxy/dialer"
)

var (
	errHijackingUnsupported = errors.New("hijacking not supported")
	errHijackingFailed      = errors.New("connection hijacking failed")
)

func (p *ProxyHandler) handleConnect(w http.ResponseWriter, r *http.Request, start time.Time, logger *zap.SugaredLogger) {
	// Hijack the connection to establish a raw TCP connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		httpError(w, http.StatusInternalServerError, errHijackingUnsupported, start, logger)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		httpError(w, http.StatusInternalServerError, errHijackingFailed, start, logger)
		return
	}

	// Connect to the target server
	ctx := getDialerContext(r)
	targetConn, err := p.connDialer.DialContext(ctx, "tcp", r.URL.Host)
	if err != nil {
		defer clientConn.Close()
		rawHttpError(clientConn, http.StatusBadGateway, err, start, logger)
		return
	}
	logger.With(
		"status", http.StatusOK,
		"latency", time.Since(start),
	).Debug("connection established")
	// Respond to the client that the tunnel is established
	_, _ = fmt.Fprint(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	// Bidirectionally copy data between client and target
	var wg sync.WaitGroup
	wg.Add(2)
	go copyAndClose(targetConn, clientConn, &wg, logger)
	go copyAndClose(clientConn, targetConn, &wg, logger)
	wg.Wait()

	logger.With(
		"status", http.StatusOK,
		"latency", time.Since(start),
	).Debug("connection closed")
}

func getDialerContext(request *http.Request) context.Context {
	proxyRequest := request.Clone(request.Context())
	// prevent sending default go user-agent
	if len(proxyRequest.Header.Values("User-Agent")) == 0 {
		proxyRequest.Header.Set("User-Agent", "")
	}
	return context.WithValue(proxyRequest.Context(), dialer.ProxyRequestKey, proxyRequest)
}

func copyAndClose(dst io.WriteCloser, src io.Reader, wg *sync.WaitGroup, logger *zap.SugaredLogger) {
	defer wg.Done()
	defer dst.Close()
	_, err := io.Copy(dst, src)
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.ErrClosedPipe) {
		logger.Warnw("error copying connection data", zap.Error(err))
	}
}

func rawHttpError(w io.Writer, status int, err error, start time.Time, logger *zap.SugaredLogger) {
	buf := new(bytes.Buffer)
	_, _ = fmt.Fprintf(buf, "HTTP/1.1 %d %s\r\n", status, http.StatusText(status))
	_, _ = fmt.Fprintf(buf, "Content-Length: %d\r\n", len(err.Error()))
	_, _ = fmt.Fprint(buf, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	_, _ = fmt.Fprint(buf, err.Error())
	_, _ = w.Write(buf.Bytes())
	logger.With(
		"status", status,
		"latency", time.Since(start),
		zap.Error(err),
	).Warn("request failed")
}
