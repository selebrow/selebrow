package kubernetes

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/selebrow/selebrow/internal/netutils"
	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/capabilities"
	"github.com/selebrow/selebrow/pkg/kubeapi"
	"github.com/selebrow/selebrow/pkg/models"
)

type KubernetesBrowserManager struct {
	cat     browsers.BrowsersCatalog
	client  kubeapi.KubernetesClient
	w       PodWatcher
	bc      BrowserConverter
	backoff wait.Backoff
	l       *zap.SugaredLogger
}

func NewKubernetesBrowserManager(
	cat browsers.BrowsersCatalog,
	client kubeapi.KubernetesClient,
	bc BrowserConverter,
	watcher PodWatcher,
	backoff wait.Backoff,
	l *zap.Logger,
) *KubernetesBrowserManager {
	kwm := &KubernetesBrowserManager{
		cat:     cat,
		client:  client,
		w:       watcher,
		bc:      bc,
		backoff: backoff,
		l:       l.Sugar(),
	}
	return kwm
}

func (m *KubernetesBrowserManager) Allocate(
	ctx context.Context,
	protocol models.BrowserProtocol,
	caps capabilities.Capabilities,
) (browser.Browser, error) {
	browserName, flavor := caps.GetName(), caps.GetFlavor()

	verCfg, ok := m.cat.LookupBrowserImage(protocol, browserName, flavor)
	if !ok {
		return nil, models.NewBadRequestError(errors.Errorf("browser %s image flavor %s is not supported", browserName, flavor))
	}

	p, err := m.createPod(ctx, verCfg, caps)
	if err != nil {
		return nil, err
	}

	m.l.Infow("pod has been created", zap.String("pod", p.Name))
	ip, err := m.w.WaitPodReady(ctx, p.Name)

	if err != nil {
		m.deletePod(context.Background(), p.Name)
		return nil, err
	}

	wd, err := m.createBrowser(p.Name, ip, verCfg, caps.IsVNCEnabled())
	if err != nil {
		m.deletePod(context.Background(), p.Name)
		return nil, err
	}

	return wd, nil
}

func (m *KubernetesBrowserManager) createPod(
	ctx context.Context,
	verCfg models.BrowserImageConfig,
	caps capabilities.Capabilities,
) (*v1.Pod, error) {
	pod, err := m.bc.ToPod(verCfg, caps)
	if err != nil {
		return nil, err
	}

	b := m.backoff
	for {
		p, err := m.client.CreatePod(ctx, &pod)
		if err == nil {
			return p, nil
		}

		if !isRetryableError(err) || b.Steps < 1 {
			return nil, err
		}

		delay := b.Step()
		m.l.With(zap.Error(err)).
			Warnf("CreatePod failed, next retry in %s, %d more tries remaining", delay.String(), b.Steps)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}

func isRetryableError(err error) bool {
	return k8sErr.IsConflict(err) ||
		k8sErr.IsTooManyRequests(err) ||
		k8sErr.IsInternalError(err) ||
		k8sErr.IsServerTimeout(err) ||
		k8sErr.IsTimeout(err)
}

func (m *KubernetesBrowserManager) deletePod(ctx context.Context, podName string) {
	l := m.l.With(zap.String("pod", podName))
	if err := m.client.DeletePod(ctx, podName); err != nil {
		l.Errorw("pod delete failed", zap.Error(err))
	} else {
		l.Infow("pod has been deleted")
	}
}

func (m *KubernetesBrowserManager) createBrowser(
	podName string,
	ip string,
	verCfg models.BrowserImageConfig,
	vncEnabled bool,
) (browser.Browser, error) {
	host := fmt.Sprintf("%s:%d", ip, verCfg.Ports[models.BrowserPort])
	forwardedHost := ip
	if !m.client.ClusterModeOut() {
		u, err := url.Parse(fmt.Sprintf("http://%s%s", host, verCfg.Path))
		if err != nil {
			return nil, err
		}

		return &kubernetesBrowser{
			forwardedHost: forwardedHost,
			u:             u,
			host:          host,
			ports:         verCfg.GetPorts(vncEnabled),
			close: func(ctx context.Context) {
				m.deletePod(ctx, podName)
			},
		}, nil
	}

	// For debug purposes only (when you run under your IDE), no test coverage expected !!!
	stopCh := make(chan struct{})
	localPorts := make(map[models.ContainerPort]int)
	var u *url.URL
	for name, port := range verCfg.GetPorts(vncEnabled) {
		p, err := netutils.FreePort()
		if err != nil {
			close(stopCh)
			return nil, err
		}

		if err := m.client.PortForwardPod(
			podName,
			int64(port), int64(p), stopCh); err != nil {
			close(stopCh)
			return nil, err
		}
		if name == models.BrowserPort {
			forwardedHost = "127.0.0.1"
			u, err = url.Parse(fmt.Sprintf("http://%s:%d%s", forwardedHost, p, verCfg.Path))
			if err != nil {
				close(stopCh)
				return nil, err
			}
		} else {
			localPorts[name] = p
		}
	}
	return &kubernetesBrowser{
		forwardedHost: forwardedHost,
		u:             u,
		host:          host,
		ports:         localPorts,
		close: func(ctx context.Context) {
			close(stopCh)
			m.deletePod(ctx, podName)
		},
	}, nil
}
