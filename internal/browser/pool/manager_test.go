package pool_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/internal/browser/pool"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

const testBrowserProtocol models.BrowserProtocol = "test"

func TestBrowserPoolManager_Allocate(t *testing.T) {
	g := NewWithT(t)

	caps := new(mocks.Capabilities)
	caps.EXPECT().GetName().Return("mosaic")

	gh := new(mocks.GetHashFunc)
	gh.EXPECT().Execute(caps).Return([]byte{0xde, 0xad})

	br1 := new(mocks.Browser)
	br2 := new(mocks.Browser)

	p := new(mocks.BrowserPool)

	f := new(mocks.BrowserPoolFactory)

	pm := pool.NewBrowserPoolManager(f, gh.Execute)

	f.EXPECT().GetPool("test-mosaic-dead").Return(p).Once()
	p.EXPECT().Checkout(context.TODO(), testBrowserProtocol, caps).Return(br1, nil).Once()

	got1, err := pm.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(got1).To(BeIdenticalTo(br1))

	p.EXPECT().Checkout(context.TODO(), testBrowserProtocol, caps).Return(br2, nil).Once()
	got2, err := pm.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(got2).To(BeIdenticalTo(br2))

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	p.EXPECT().Shutdown(ctx).Return(nil).Once()
	err = pm.Shutdown(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = pm.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(HaveOccurred())
}
