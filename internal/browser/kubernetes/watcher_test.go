package kubernetes

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/selebrow/selebrow/mocks"
)

func TestPodWatcherImpl_WaitPodReady(t *testing.T) {
	g := NewWithT(t)

	client := new(mocks.KubernetesClient)
	ch := make(chan *watch.Event)
	client.EXPECT().Watch(mock.Anything, &metav1.LabelSelector{MatchLabels: map[string]string{"lineage": "123"}}).Return(ch, nil)
	w, err := NewPodWatcher(client, "123", zaptest.NewLogger(t))
	g.Expect(err).ToNot(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	go func() {
		ch <- &watch.Event{
			Type:   watch.Added,
			Object: makePod("mypod1", "", false),
		}

		ch <- &watch.Event{
			Type:   watch.Modified,
			Object: makePod("mypod1", "11.22.33.44", false),
		}

		ch <- &watch.Event{
			Type:   watch.Modified,
			Object: makePod("mypod1", "11.22.33.44", true),
		}

		ch <- &watch.Event{
			Type:   watch.Modified,
			Object: makePod("mypod1", "11.22.33.44", true),
		}
	}()

	ip, err := w.WaitPodReady(ctx, "mypod1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ip).To(Equal("11.22.33.44"))

	go func() {
		ch <- &watch.Event{
			Type:   watch.Added,
			Object: makePod("mypod2", "", false),
		}

		ch <- &watch.Event{
			Type:   watch.Deleted,
			Object: makePod("mypod2", "", false),
		}
	}()

	_, err = w.WaitPodReady(ctx, "mypod2")
	g.Expect(err).To(HaveOccurred())

	client.AssertExpectations(t)
}

func TestPodWatcherImpl_Shutdown(t *testing.T) {
	g := NewWithT(t)

	client := new(mocks.KubernetesClient)
	ch := make(chan *watch.Event)
	var ctx context.Context
	client.EXPECT().Watch(mock.Anything, mock.Anything).Run(func(c context.Context, _ *metav1.LabelSelector) {
		g.Expect(c).ToNot(BeNil())
		ctx = c
	}).Return(ch, nil)

	w, err := NewPodWatcher(client, "111", zaptest.NewLogger(t))
	g.Expect(err).ToNot(HaveOccurred())

	go func() {
		<-ctx.Done()
		close(ch)
	}()

	err = w.Shutdown(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())
}

func makePod(name, ip string, ready bool) *v1.Pod {
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "maybebrowser",
				},
			},
		},
		Status: v1.PodStatus{PodIP: ip},
	}

	if ip != "" {
		p.Status.ContainerStatuses = []v1.ContainerStatus{
			{Ready: ready},
		}
	}

	return p
}
