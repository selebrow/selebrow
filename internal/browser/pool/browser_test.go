package pool_test

import (
	"context"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/internal/browser/pool"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

func TestPooledBrowser_GetURL(t *testing.T) {
	g := NewWithT(t)

	br := new(mocks.Browser)
	u, err := url.Parse("http://example.com")
	g.Expect(err).ToNot(HaveOccurred())
	br.EXPECT().GetURL().Return(u).Once()

	pbr := pool.NewPooledBrowser(br, nil)

	got := pbr.GetURL()
	g.Expect(*got).To(Equal(*u))
}

func TestPooledBrowser_GetHost(t *testing.T) {
	g := NewWithT(t)

	br := new(mocks.Browser)
	br.EXPECT().GetHost().Return("qqq").Once()

	pbr := pool.NewPooledBrowser(br, nil)

	got := pbr.GetHost()
	g.Expect(got).To(Equal("qqq"))
}

func TestPooledBrowser_GetHostPort(t *testing.T) {
	g := NewWithT(t)

	br := new(mocks.Browser)
	br.EXPECT().GetHostPort(models.ClipboardPort).Return("none:666")

	pbr := pool.NewPooledBrowser(br, nil)

	got := pbr.GetHostPort(models.ClipboardPort)
	g.Expect(got).To(Equal("none:666"))
}

func TestPooledBrowser_Close(t *testing.T) {
	g := NewWithT(t)
	br := new(mocks.Browser)

	var called bool
	pbr := pool.NewPooledBrowser(br, func(_ *pool.PooledBrowser) {
		called = true
	})

	pbr.Close(context.TODO(), false)
	g.Expect(called).To(BeTrue())
}

func TestPooledBrowser_CloseTrash(t *testing.T) {
	br := new(mocks.Browser)

	pbr := pool.NewPooledBrowser(br, nil)

	br.EXPECT().Close(context.TODO(), true).Once()

	pbr.Close(context.TODO(), true)
}
