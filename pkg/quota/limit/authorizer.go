package limit

import (
	"container/list"
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/selebrow/selebrow/pkg/models"
)

type elementValue chan struct{}

type LimitQuotaAuthorizer struct {
	limit     int
	allocated int
	m         sync.RWMutex
	queue     *list.List
	qLimit    int
	l         *zap.SugaredLogger
}

func NewLimitQuotaAuthorizer(limit, qLimit int, l *zap.Logger) *LimitQuotaAuthorizer {
	logger := l.Sugar()
	logger.Infow("initializing quota", zap.Int("limit", limit), zap.Int("queue_limit", qLimit))
	return &LimitQuotaAuthorizer{
		limit:  limit,
		queue:  list.New(),
		qLimit: qLimit,
		l:      logger,
	}
}

func (q *LimitQuotaAuthorizer) Enabled() bool {
	return q != nil
}

func (q *LimitQuotaAuthorizer) ExternalReserve(qty int) int {
	q.m.Lock()
	defer q.m.Unlock()
	q.allocated += qty
	return q.allocated
}

func (q *LimitQuotaAuthorizer) Reserve(ctx context.Context) error {
	q.m.Lock()
	qSize := q.queue.Len()
	// fast path only when we have enough quota to accommodate new request + queued requests (if any)
	// this is to avoid granting quota to the new request before any pending requests
	if q.allocated+qSize < q.limit {
		defer q.m.Unlock()
		q.allocated++
		q.l.Debugf("quota reserved: allocated=%d", q.allocated)
		return nil
	}

	if qSize >= q.qLimit {
		defer q.m.Unlock()
		return models.NewQuoteExceededError(errors.New(q.formatError("quota exceeded")))
	}

	ch := make(elementValue)
	e := q.queue.PushBack(ch)
	q.m.Unlock()

	select {
	case <-ctx.Done():
		q.m.Lock()
		defer q.m.Unlock()
		select {
		case <-ch:
			// we've got quota at the last moment and element was removed from the waiting queue in Release()
			return nil
		default:
			q.queue.Remove(e)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return models.NewQuoteExceededError(errors.Wrap(ctx.Err(), q.formatError("quota wait failed")))
			} else {
				return errors.Wrap(ctx.Err(), q.formatError("quota wait cancelled"))
			}
		}
	case <-ch:
		return nil
	}
}

func (q *LimitQuotaAuthorizer) Release() int {
	q.m.Lock()
	defer q.m.Unlock()

	if e := q.queue.Front(); e != nil {
		ch, _ := q.queue.Remove(e).(elementValue)
		q.l.Debugf("quota reserved by queue: allocated=%d, queue size=%d", q.allocated, q.queue.Len())
		close(ch)
		return q.allocated
	}

	if q.allocated < 1 {
		q.l.Warnf("quota underrun detected, resetting to 0: allocated=%d", q.allocated)
		q.allocated = 0
	} else {
		q.allocated--
		q.l.Debugf("quota released: allocated=%d", q.allocated)
	}
	return q.allocated
}

func (q *LimitQuotaAuthorizer) Limit() int {
	return q.limit
}

func (q *LimitQuotaAuthorizer) Allocated() int {
	q.m.RLock()
	defer q.m.RUnlock()
	return q.allocated
}

func (q *LimitQuotaAuthorizer) QueueLimit() int {
	return q.qLimit
}

func (q *LimitQuotaAuthorizer) QueueSize() int {
	q.m.RLock()
	defer q.m.RUnlock()
	return q.queue.Len()
}

func (q *LimitQuotaAuthorizer) formatError(msg string) string {
	return fmt.Sprintf("%s: allocated=%d, limit=%d, queue size=%d", msg, q.allocated, q.limit, q.queue.Len())
}
