package kubernetes_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/selebrow/selebrow/internal/browser/kubernetes"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

const testBrowserProtocol models.BrowserProtocol = "test"

func TestKubernetesBrowserManager_Allocate(t *testing.T) {
	g := NewWithT(t)

	cfg := models.BrowserImageConfig{
		Image: "apple/safari",
		Ports: map[models.ContainerPort]int{
			models.BrowserPort:   123,
			models.ClipboardPort: 777,
			models.VNCPort:       444,
		},
		Path: "/wd",
		Env:  map[string]string{"a": "b", "b": "c"},
		Limits: map[string]string{
			"cpu": "1",
			"mem": "100500Gi",
		},
	}
	cat := new(mocks.BrowsersCatalog)
	cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(cfg, true)

	client := new(mocks.KubernetesClient)
	client.EXPECT().ClusterModeOut().Return(false)

	bc := new(mocks.BrowserConverter)
	w := new(mocks.PodWatcher)

	mgr := kubernetes.NewKubernetesBrowserManager(cat, client, bc, w, wait.Backoff{}, zaptest.NewLogger(t))

	caps := createCaps("safari", "135", "def", false)

	pod := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "mypod"}}
	bc.EXPECT().ToPod(cfg, caps).Return(pod, nil)
	client.EXPECT().CreatePod(context.TODO(), &pod).Return(&pod, nil).Once()
	w.EXPECT().WaitPodReady(context.TODO(), "mypod").Return("1.2.3.4", nil).Once()
	wd, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())

	u := wd.GetURL()
	g.Expect(u.String()).To(Equal("http://1.2.3.4:123/wd"))
	g.Expect(wd.GetHost()).To(Equal("1.2.3.4:123"))

	g.Expect(wd.GetHostPort(models.ClipboardPort)).To(Equal("1.2.3.4:777"))
	g.Expect(wd.GetHostPort(models.VNCPort)).To(BeEmpty())

	client.EXPECT().DeletePod(context.TODO(), "mypod").Return(nil).Once()
	wd.Close(context.TODO(), true)

	bc.AssertExpectations(t)
	client.AssertExpectations(t)
	w.AssertExpectations(t)
}

func TestKubernetesBrowserManager_AllocateNoBrowser(t *testing.T) {
	g := NewWithT(t)

	cat := createBrowserCatalog("safari")

	mgr := kubernetes.NewKubernetesBrowserManager(cat, nil, nil, nil, wait.Backoff{}, zaptest.NewLogger(t))

	caps1 := createCaps("chrome", "199+", "def", false)

	_, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps1)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusBadRequest))

	caps2 := createCaps("safari", "136", "qqq", false)

	_, err = mgr.Allocate(context.TODO(), testBrowserProtocol, caps2)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusBadRequest))
}

func TestKubernetesBrowserManager_AllocatePodConversionError(t *testing.T) {
	g := NewWithT(t)

	cat := createBrowserCatalog("safari")
	bc := new(mocks.BrowserConverter)

	mgr := kubernetes.NewKubernetesBrowserManager(cat, nil, bc, nil, wait.Backoff{}, zaptest.NewLogger(t))

	caps := createCaps("safari", "135", "def", true)

	bc.EXPECT().ToPod(models.BrowserImageConfig{}, caps).Return(v1.Pod{}, errors.New("test pod conversion error"))

	_, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError("test pod conversion error"))
}

func TestKubernetesBrowserManager_AllocateCreatePodError(t *testing.T) {
	g := NewWithT(t)

	cat := createBrowserCatalog("safari")
	bc := new(mocks.BrowserConverter)
	client := new(mocks.KubernetesClient)

	mgr := kubernetes.NewKubernetesBrowserManager(cat, client, bc, nil, wait.Backoff{}, zaptest.NewLogger(t))

	caps := createCaps("safari", "135", "def", false)

	pod := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nocreate"}}
	bc.EXPECT().ToPod(models.BrowserImageConfig{}, caps).Return(pod, nil)
	client.EXPECT().CreatePod(context.TODO(), &pod).Return(&pod, errors.New("test pod create error")).Once()
	_, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError("test pod create error"))

	client.AssertExpectations(t)
}

func TestKubernetesBrowserManager_AllocateCreatePodError_Backoff(t *testing.T) {
	g := NewWithT(t)

	cat := createBrowserCatalog("safari")
	bc := new(mocks.BrowserConverter)
	client := new(mocks.KubernetesClient)

	mgr := kubernetes.NewKubernetesBrowserManager(cat, client, bc, nil, wait.Backoff{
		Duration: 50 * time.Millisecond,
		Factor:   2,
		Steps:    1,
	}, zaptest.NewLogger(t))

	caps := createCaps("safari", "135", "def", false)

	pod := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nocreate"}}
	bc.EXPECT().ToPod(models.BrowserImageConfig{}, caps).Return(pod, nil)
	client.EXPECT().CreatePod(context.TODO(), &pod).Return(nil, k8sErr.NewInternalError(errors.New("test pod retryable error"))).Twice()
	_, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError(MatchRegexp("test pod retryable error")))

	client.AssertExpectations(t)
}

func TestKubernetesBrowserManager_AllocatePodWatchError(t *testing.T) {
	g := NewWithT(t)

	cat := createBrowserCatalog("chrome")
	bc := new(mocks.BrowserConverter)
	client := new(mocks.KubernetesClient)
	w := new(mocks.PodWatcher)

	mgr := kubernetes.NewKubernetesBrowserManager(cat, client, bc, w, wait.Backoff{}, zaptest.NewLogger(t))

	caps := createCaps("chrome", "135", "def", false)

	pod := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nostart"}}
	bc.EXPECT().ToPod(models.BrowserImageConfig{}, caps).Return(pod, nil)
	client.EXPECT().CreatePod(context.TODO(), &pod).Return(&pod, nil).Once()
	w.EXPECT().WaitPodReady(context.TODO(), "nostart").Return("", errors.New("test pod watch error")).Once()
	client.EXPECT().DeletePod(context.Background(), "nostart").Return(nil).Once()
	_, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).To(MatchError("test pod watch error"))

	client.AssertExpectations(t)
	w.AssertExpectations(t)
}

func createBrowserCatalog(name string) *mocks.BrowsersCatalog {
	cat := new(mocks.BrowsersCatalog)
	cat.EXPECT().LookupBrowserImage(testBrowserProtocol, name, "def").Return(models.BrowserImageConfig{}, true)
	cat.EXPECT().LookupBrowserImage(testBrowserProtocol, mock.Anything, mock.Anything).Return(models.BrowserImageConfig{}, false)

	return cat
}

func createCaps(name, version, flavor string, vncEnabled bool) *mocks.Capabilities {
	caps := new(mocks.Capabilities)
	caps.EXPECT().GetName().Return(name)
	caps.EXPECT().GetVersion().Return(version)
	caps.EXPECT().IsVNCEnabled().Return(vncEnabled)
	caps.EXPECT().GetFlavor().Return(flavor)
	return caps
}
