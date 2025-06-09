package netutils

import "net"

// FreePort finds random available port to listen on
func FreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	listenAddr, _ := l.Addr().(*net.TCPAddr)

	return listenAddr.Port, nil
}
