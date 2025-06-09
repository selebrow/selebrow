package kubeapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/portforward"
	clientWatch "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/transport/spdy"
)

type (
	PortForwardAPodRequest struct {
		// RestConfig is the kubernetes config
		RestConfig *rest.Config
		// Pod is the selected pod for this port forwarding
		Pod core.Pod
		// LocalPort is the local port that will be selected to expose the PodPort
		LocalPort int
		// PodPort is the target port for the pod
		PodPort int
		// Steams configures where to write or read input from
		Streams IOStreams
		// StopCh is the channel used to manage the port forward lifecycle
		StopCh chan struct{}
		// ReadyCh communicates when the tunnel is ready to receive traffic
		ReadyCh chan struct{}
	}

	IOStreams struct {
		// In think, os.Stdin
		In io.Reader
		// Out think, os.Stdout
		Out io.Writer
		// ErrOut think, os.Stderr
		ErrOut io.Writer
	}
)

func (c *Client) CreatePod(ctx context.Context, pod *core.Pod) (*core.Pod, error) {
	podsClient := c.clientset.CoreV1().Pods(c.namespace)

	pod, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})

	if err != nil {
		return nil, err
	}

	return pod, err
}

func (c *Client) ListPods(ctx context.Context, labelSelector *metav1.LabelSelector) (*core.PodList, error) {
	podsClient := c.clientset.CoreV1().Pods(c.namespace)

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	pods, err := podsClient.List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	return pods, nil
}

func (c *Client) DeletePod(ctx context.Context, name string) error {
	podsClient := c.clientset.CoreV1().Pods(c.namespace)

	err := podsClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Watch(ctx context.Context, labelSelector *metav1.LabelSelector) (<-chan *watch.Event, error) {
	podsClient := c.clientset.CoreV1().Pods(c.namespace)

	var tm int64 = 3600 * 4
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	opts := metav1.ListOptions{
		LabelSelector:  selector.String(),
		TimeoutSeconds: &tm,
	}

	watcher, err := clientWatch.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
			return podsClient.Watch(ctx, opts)
		},
	})
	if err != nil {
		return nil, err
	}

	events := make(chan *watch.Event)

	go func() {
		defer watcher.Stop()
		defer close(events)
		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				events <- &event
			case <-watcher.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return events, nil
}

func (c *Client) PortForwardPod(podName string, podPort, localport int64, stopCh chan struct{}) error {
	// readyCh marks when the port is ready to get traffic.
	readyCh := make(chan struct{})
	stream := IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	return c.portForward(&PortForwardAPodRequest{
		RestConfig: c.restConfig,
		LocalPort:  int(localport),
		PodPort:    int(podPort),
		Streams:    stream,
		StopCh:     stopCh,
		ReadyCh:    readyCh,
		Pod: core.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: c.namespace,
			},
		},
	})
}

func (c *Client) portForward(request *PortForwardAPodRequest) error {
	if request.RestConfig == nil {
		return fmt.Errorf("request host config is nil, cannot forward ports")
	}

	go func() {
		for {
			select {
			case <-request.StopCh:
				return
			default:
				err := portForwardAPod(request)
				if err != nil {
					c.l.Errorw("PortForward failed", zap.Error(err))
					return
				}
				request.ReadyCh = make(chan struct{})
			}
		}
	}()

	<-request.ReadyCh

	return nil
}

func portForwardAPod(req *PortForwardAPodRequest) error {
	path := fmt.Sprintf(
		"/api/v1/namespaces/%s/pods/%s/portforward",
		req.Pod.Namespace, req.Pod.Name,
	)

	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{Scheme: "https", Path: path, Host: hostIP},
	)

	fw, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)},
		req.StopCh,
		req.ReadyCh,
		req.Streams.Out,
		req.Streams.ErrOut,
	)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}
