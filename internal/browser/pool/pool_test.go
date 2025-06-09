package pool_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/browser/pool"
	"github.com/selebrow/selebrow/mocks"
)

func TestIdleBrowserPool_CheckoutReuse(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(50 * time.Millisecond)
	cfg.EXPECT().MaxAge().Return(2 * time.Second)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	u1, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())
	br1 := new(mocks.Browser)
	br1.EXPECT().GetURL().Return(u1)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()
	got1, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got1.GetURL()).To(Equal(u1))

	got1.Close(context.TODO(), false)

	got2, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got2.GetURL()).To(Equal(u1))

	g.Expect(p.Shutdown(context.TODO())).To(Succeed())
}

func TestIdleBrowserPool_CheckoutError(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(50 * time.Millisecond)
	cfg.EXPECT().MaxAge().Return(2 * time.Second)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(nil, errors.New("fake error")).Once()
	_, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError("fake error"))
	size, _ := p.PoolState()
	g.Expect(size).To(BeZero())

	g.Expect(p.Shutdown(context.TODO())).To(Succeed())
}

func TestIdleBrowserPool_CheckoutEvictIdle(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(50 * time.Millisecond)
	cfg.EXPECT().MaxAge().Return(2 * time.Second)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	u1, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())
	br1 := new(mocks.Browser)
	br1.EXPECT().GetURL().Return(u1)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()
	got1, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got1.GetURL()).To(Equal(u1))

	closed := make(chan struct{})
	br1.EXPECT().Close(context.Background(), true).Run(func(_ context.Context, _ bool) {
		close(closed)
	}).Once()

	got1.Close(context.TODO(), false)
	g.Eventually(closed).Should(BeClosed())
	size, _ := p.PoolState()
	g.Expect(size).To(BeZero())

	g.Expect(p.Shutdown(context.TODO())).To(Succeed())
}

func TestIdleBrowserPool_CheckoutMaxIdle(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(1 * time.Second)
	cfg.EXPECT().MaxAge().Return(2 * time.Second)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	u1, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())
	br1 := new(mocks.Browser)
	br1.EXPECT().GetURL().Return(u1)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()
	got1, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got1.GetURL()).To(Equal(u1))

	u2, err := url.Parse("http://host2")
	g.Expect(err).ToNot(HaveOccurred())
	br2 := new(mocks.Browser)
	br2.EXPECT().GetURL().Return(u2)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br2, nil).Once()
	got2, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got2.GetURL()).To(Equal(u2))

	got1.Close(context.TODO(), false)

	br2.EXPECT().Close(context.Background(), true).Once()
	got2.Close(context.TODO(), false)
	br2.AssertExpectations(t)

	// Only br1 should have returned to pool
	size, _ := p.PoolState()
	g.Expect(size).To(Equal(1))

	got3, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got3.GetURL()).To(Equal(u1))
	br1.AssertExpectations(t)

	g.Expect(p.Shutdown(context.TODO())).To(Succeed())
}

func TestIdleBrowserPool_CheckoutMaxAge(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(2 * time.Second)
	cfg.EXPECT().MaxAge().Return(50 * time.Millisecond)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	u1, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())
	br1 := new(mocks.Browser)
	br1.EXPECT().GetURL().Return(u1)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()
	got1, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got1.GetURL()).To(Equal(u1))

	u2, err := url.Parse("http://host2")
	g.Expect(err).ToNot(HaveOccurred())
	br2 := new(mocks.Browser)
	br2.EXPECT().GetURL().Return(u2)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br2, nil).Once()
	got2, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got2.GetURL()).To(Equal(u2))

	closed := make(chan struct{})
	br2.EXPECT().Close(context.Background(), true).Run(func(_ context.Context, _ bool) {
		close(closed)
	}).Once()
	got2.Close(context.TODO(), false)
	g.Eventually(closed).Should(BeClosed())
	br2.AssertExpectations(t)
	size, _ := p.PoolState()
	g.Expect(size).Should(BeZero())

	br1.EXPECT().Close(context.Background(), true).Once()
	got1.Close(context.TODO(), false)
	br1.AssertExpectations(t)
	size, _ = p.PoolState()
	g.Expect(size).Should(BeZero(), "it should have not returned to the pool because of maxAge and it was older than br2")

	g.Expect(p.Shutdown(context.TODO())).To(Succeed())
}

func TestIdleBrowserPool_Shutdown(t *testing.T) {
	g := NewWithT(t)

	mgr := new(mocks.BrowserManager)
	cfg := new(mocks.PoolConfig)

	cfg.EXPECT().IdleTimeout().Return(1 * time.Second)
	cfg.EXPECT().MaxAge().Return(1 * time.Second)
	cfg.EXPECT().MaxIdle().Return(1)
	p := pool.NewIdleBrowserPool("abc", mgr, cfg, zaptest.NewLogger(t))

	caps := new(mocks.Capabilities)

	u1, err := url.Parse("http://host1")
	g.Expect(err).ToNot(HaveOccurred())
	br1 := new(mocks.Browser)
	br1.EXPECT().GetURL().Return(u1)

	mgr.EXPECT().Allocate(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()
	got1, err := p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(got1.GetURL()).To(Equal(u1))

	got1.Close(context.TODO(), false)

	br1.EXPECT().Close(context.TODO(), true).Once()
	err = p.Shutdown(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())

	br1.AssertExpectations(t)

	size, shut := p.PoolState()
	g.Expect(size).To(BeZero())
	g.Expect(shut).To(BeTrue())

	_, err = p.Checkout(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError(MatchRegexp(`.*shutdown.*`)))
}
