package conn

import (
	"context"
	"net"
)

type ConnFactory interface {
	GetConn(ctx context.Context, hostport string) (net.Conn, error)
}

type TcpConnFactory struct{}

func (t *TcpConnFactory) GetConn(ctx context.Context, hostport string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", hostport)
}
