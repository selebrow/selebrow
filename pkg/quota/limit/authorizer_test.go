package limit

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

func TestLimitQuotaAuthorizer(t *testing.T) {
	g := NewWithT(t)
	q := NewLimitQuotaAuthorizer(2, 0, zaptest.NewLogger(t))

	g.Expect(q.Enabled()).To(BeTrue())
	g.Expect(q.Limit()).To(Equal(2))

	err := q.Reserve(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(q.Allocated()).To(Equal(1))

	err = q.Reserve(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(q.Allocated()).To(Equal(2))

	err = q.Reserve(context.TODO())
	g.Expect(err).To(HaveOccurred())
	g.Expect(q.Allocated()).To(Equal(2))

	got := q.ExternalReserve(1)
	g.Expect(got).To(Equal(3))
	g.Expect(q.Allocated()).To(Equal(3))

	got = q.Release()
	g.Expect(got).To(Equal(2))
	g.Expect(q.Allocated()).To(Equal(2))

	got = q.Release()
	g.Expect(got).To(Equal(1))
	g.Expect(q.Allocated()).To(Equal(1))

	got = q.Release()
	g.Expect(got).To(Equal(0))
	g.Expect(q.Allocated()).To(Equal(0))

	got = q.Release()
	g.Expect(got).To(Equal(0))
	g.Expect(q.Allocated()).To(Equal(0))
}

func TestLimitQuotaAuthorizer_Queue(t *testing.T) {
	g := NewWithT(t)
	q := NewLimitQuotaAuthorizer(1, 2, zaptest.NewLogger(t))

	g.Expect(q.Enabled()).To(BeTrue())
	g.Expect(q.Limit()).To(Equal(1))
	g.Expect(q.QueueLimit()).To(Equal(2))

	err := q.Reserve(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(q.Allocated()).To(Equal(1))
	g.Expect(q.QueueSize()).To(Equal(0))

	ctx, cancel := context.WithCancel(context.TODO())

	ch := make(chan error)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ch <- q.Reserve(ctx)
	}()
	go func() {
		defer wg.Done()
		ch <- q.Reserve(ctx)
	}()

	g.Eventually(q.QueueSize).Should(Equal(2))
	got := q.Release()
	g.Expect(got).To(Equal(1)) // preempted by queue
	g.Eventually(ch).Should(Receive(&err))
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(q.Allocated()).To(Equal(1))
	g.Expect(q.QueueSize()).To(Equal(1))

	cancel()
	g.Eventually(ch).Should(Receive(&err))
	g.Expect(err).To(MatchError(context.Canceled))
	g.Expect(q.Allocated()).To(Equal(1))
	g.Expect(q.QueueSize()).To(Equal(0))

	tCtx, tCancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
	defer tCancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- q.Reserve(tCtx)
	}()
	g.Eventually(ch).Should(Receive(&err))
	g.Expect(err).To(MatchError(context.DeadlineExceeded))
	g.Expect(q.Allocated()).To(Equal(1))
	g.Expect(q.QueueSize()).To(Equal(0))

	wg.Wait()
}
