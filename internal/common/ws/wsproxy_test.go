package ws

import (
	"net"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/posener/wstest"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/mocks"
)

func TestWSProxyImpl_Handler_Serve(t *testing.T) {
	g := NewWithT(t)
	cf := new(mocks.ConnFactory)
	p := NewWSProxyImpl(cf, zaptest.NewLogger(t))
	h := p.Handler("wshost:1234")
	d := wstest.NewDialer(h)

	clconn, srvconn := net.Pipe()
	cf.EXPECT().GetConn(mock.Anything, "wshost:1234").Return(clconn, nil)

	hdr := make(http.Header)
	hdr.Set("Origin", "http://testclient")
	wc, r, err := d.Dial("ws://ignored/wsendpoint", hdr)
	g.Expect(err).ToNot(HaveOccurred())
	defer wc.Close()
	defer r.Body.Close()
	g.Expect(r).To(HaveHTTPStatus("101 Switching Protocols"))

	test := "testmsg"
	err = wc.WriteMessage(websocket.TextMessage, []byte(test))
	g.Expect(err).ToNot(HaveOccurred())

	var got = make([]byte, len(test))
	_, err = srvconn.Read(got)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(got)).To(Equal(test))

	ch := make(chan struct{})
	wc.SetCloseHandler(func(code int, text string) error {
		close(ch)
		return nil
	})

	srvconn.Close()
	_, _, err = wc.ReadMessage()
	g.Expect(err).To(BeAssignableToTypeOf(&websocket.CloseError{}))
	g.Eventually(ch).Should(BeClosed())
}

func TestWSProxyImpl_Handler_Serve_ConnError(t *testing.T) {
	g := NewWithT(t)
	cf := new(mocks.ConnFactory)
	p := NewWSProxyImpl(cf, zaptest.NewLogger(t))
	h := p.Handler("wshost:1234")
	d := wstest.NewDialer(h)

	cf.EXPECT().GetConn(mock.Anything, "wshost:1234").Return(nil, errors.New("test conn error"))

	hdr := make(http.Header)
	hdr.Set("Origin", "http://testclient")
	wc, r, err := d.Dial("ws://ignored/wsendpoint", hdr)
	g.Expect(err).ToNot(HaveOccurred())
	defer wc.Close()
	defer r.Body.Close()
	g.Expect(r).To(HaveHTTPStatus("101 Switching Protocols"))

	ch := make(chan struct{})
	wc.SetCloseHandler(func(code int, text string) error {
		close(ch)
		return nil
	})
	_, _, err = wc.ReadMessage()
	g.Expect(err).To(BeAssignableToTypeOf(&websocket.CloseError{}))
	g.Eventually(ch).Should(BeClosed())
}
