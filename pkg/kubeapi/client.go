package kubeapi

import (
	"context"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/selebrow/selebrow/pkg/config"
)

type KubernetesClient interface {
	ClusterModeOut() bool
	CreatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error)
	ListPods(ctx context.Context, selector *metav1.LabelSelector) (*v1.PodList, error)
	DeletePod(ctx context.Context, name string) error
	Watch(ctx context.Context, selector *metav1.LabelSelector) (<-chan *watch.Event, error)
	PortForwardPod(podName string, podPort, localport int64, stopCh chan struct{}) error
}

type ProxyFunc func(*http.Request) (*url.URL, error)

type Client struct {
	clientset      kubernetes.Interface
	namespace      string
	restConfig     *rest.Config
	clusterModeOut bool
	l              *zap.SugaredLogger
}

func (c *Client) ClusterModeOut() bool {
	return c.clusterModeOut
}

func NewClient(cfg config.KubeConfig, l *zap.Logger) (*Client, error) {
	var (
		client  *Client
		err     error
		noProxy = func(*http.Request) (*url.URL, error) {
			return nil, nil
		}
	)

	if cfg.KubeClusterModeOut() {
		client, err = getDefaultOutOfClusterClient(cfg.KubeConfig(), noProxy)
	} else {
		client, err = getDefaultInClusterClient(noProxy)
	}

	if err != nil {
		return nil, err
	}

	client.namespace = cfg.Namespace()
	client.l = l.Sugar()
	return client, nil
}

func getDefaultOutOfClusterClient(kubeconfig string, proxyFn ProxyFunc) (*Client, error) {
	// use the current context in kubeconfig
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	cfg.Proxy = proxyFn
	// create the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset:      clientset,
		restConfig:     cfg,
		clusterModeOut: true,
	}, nil
}

func getDefaultInClusterClient(proxyFn ProxyFunc) (*Client, error) {
	// creates the in-cluster config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	cfg.Proxy = proxyFn
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset:  clientset,
		restConfig: cfg,
	}, nil
}
