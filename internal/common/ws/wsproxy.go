package ws

import (
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"

	"github.com/selebrow/selebrow/internal/common/conn"
)

type WSProxy interface {
	Handler(hostport string) websocket.Handler
}

type WSProxyImpl struct {
	connfactory conn.ConnFactory
	l           *zap.SugaredLogger
}

func NewWSProxyImpl(connfactory conn.ConnFactory, l *zap.Logger) *WSProxyImpl {
	return &WSProxyImpl{
		connfactory: connfactory,
		l:           l.Sugar(),
	}
}

func (w *WSProxyImpl) Handler(hostport string) websocket.Handler {
	return func(wsconn *websocket.Conn) {
		cn, err := w.connfactory.GetConn(wsconn.Request().Context(), hostport)
		if err != nil {
			w.l.Errorw("WS connection error", zap.Error(err))
			wsconn.Close()
			return
		}
		var wg sync.WaitGroup
		wg.Add(2)
		wsconn.PayloadType = websocket.BinaryFrame
		go func() {
			defer wg.Done()
			defer wsconn.Close()
			_, err = io.Copy(wsconn, cn)
			if logCopyErr(err) {
				w.l.Errorw("WS Proxy error", zap.Error(err))
			}
			w.l.Info("WS session closed")
		}()

		go func() {
			defer wg.Done()
			defer cn.Close()
			_, err = io.Copy(cn, wsconn)
			if logCopyErr(err) {
				w.l.Errorw("WS Proxy error", zap.Error(err))
			}
		}()
		wg.Wait()
	}
}

func logCopyErr(err error) bool {
	// net.ErrClosed is valid case when client closes connection, io.ErrClosedPipe - same in tests we have to exclude it
	// to avoid race condition with goroutine using logger when actual test is complete
	return err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.ErrClosedPipe)
}
