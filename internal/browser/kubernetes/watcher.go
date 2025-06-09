package kubernetes

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/selebrow/selebrow/pkg/kubeapi"
)

type PodWatcher interface {
	WaitPodReady(ctx context.Context, podName string) (string, error)
}

type PodWatcherImpl struct {
	client  kubeapi.KubernetesClient
	mtx     sync.RWMutex
	waiters map[string]chan string
	cancel  context.CancelFunc
	done    chan struct{}
	l       *zap.SugaredLogger
}

func NewPodWatcher(client kubeapi.KubernetesClient, lineage string, l *zap.Logger) (*PodWatcherImpl, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &PodWatcherImpl{
		client:  client,
		waiters: make(map[string]chan string),
		cancel:  cancel,
		done:    make(chan struct{}),
		l:       l.Sugar(),
	}
	sel := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			kubeapi.LineageLabel: lineage,
		},
	}

	events, err := w.client.Watch(ctx, sel)
	if err != nil {
		return nil, err
	}
	go w.watchPodEvents(events)
	return w, nil
}

func (w *PodWatcherImpl) WaitPodReady(ctx context.Context, podName string) (string, error) {
	ch := w.addWaiter(podName)
	var (
		ip  string
		err error
		ok  bool
	)

	select {
	case ip, ok = <-ch:
	case <-ctx.Done():
		err = ctx.Err()
	}

	w.removeWaiter(podName)
	if err == nil && !ok {
		return "", errors.New("watcher was closed")
	}
	return ip, err
}

func (w *PodWatcherImpl) Shutdown(ctx context.Context) error {
	w.l.Info("pod watcher is shutting down...")
	w.cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.done:
	}
	return nil
}

func (w *PodWatcherImpl) watchPodEvents(events <-chan *watch.Event) {
	podsReady := make(map[string]bool)
	for e := range events {
		pod, ok := e.Object.(*core.Pod)
		if !ok {
			continue
		}
		switch e.Type {
		case watch.Added:
			podsReady[pod.Name] = false
		case watch.Modified:
			if podsReady[pod.Name] || len(pod.Status.ContainerStatuses) != len(pod.Spec.Containers) {
				break
			}
			ready := 0
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					break
				}
				ready++
			}
			if ready == len(pod.Spec.Containers) {
				podsReady[pod.Name] = true
				w.notifyWaiter(pod.Name, pod.Status.PodIP)
			}
		case watch.Deleted:
			delete(podsReady, pod.Name)
			w.removeWaiter(pod.Name)
		default:
		}
	}

	w.l.Info("pod events watcher shutdown completed")
	close(w.done)
}

func (w *PodWatcherImpl) notifyWaiter(podName, ip string) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	ch, ok := w.waiters[podName]
	if ok {
		ch <- ip
	}
}

func (w *PodWatcherImpl) removeWaiter(podName string) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	ch, ok := w.waiters[podName]
	if ok {
		close(ch)
		delete(w.waiters, podName)
	}
}

func (w *PodWatcherImpl) addWaiter(podName string) <-chan string {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	ch := make(chan string, 1)
	w.waiters[podName] = ch
	return ch
}
