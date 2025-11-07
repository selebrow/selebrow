package app

import (
	"os"
	"path"
	"time"

	"github.com/selebrow/selebrow/internal/browser/kubernetes"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/config"
	"github.com/selebrow/selebrow/pkg/kubeapi"
	"github.com/selebrow/selebrow/pkg/log"
	"github.com/selebrow/selebrow/pkg/quota"
	"github.com/selebrow/selebrow/pkg/signal"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

func InitKubeClientFunc(cfg config.Config) kubeapi.KubernetesClient {
	l := log.GetLogger().Named("k8s").Named("client")
	kubeClient, err := kubeapi.NewClient(cfg, l)
	if err != nil {
		InitLog.Fatalw("failed to initialize Kubernetes client", zap.Error(err))
	}

	return kubeClient
}

func InitKubernetesQuotaAuthorizerFunc(cfg config.Config, _ kubeapi.KubernetesClient, _ *signal.Handler) quota.QuotaAuthorizer {
	// XXX maybe calculate from kube ResourceQuota?
	return initLimitQuotaAuthorizer(cfg, 0, 0)
}

func readKubeTemplates(cfg config.Config) map[string]string {
	templatesData := make(map[string]string)
	templatesPath := cfg.KubeTemplatesPath()

	for _, fn := range templateFiles {
		data, err := os.ReadFile(path.Join(templatesPath, fn))
		if err != nil {
			InitLog.Fatalw("failed to read template file", zap.Error(err), zap.String("name", fn))
		}
		templatesData[fn] = string(data)
	}

	return templatesData
}

func initKubernetesWebDriverManager(
	cfg config.Config,
	client kubeapi.KubernetesClient,
	templatesData map[string]string,
	cat browsers.BrowsersCatalog,
	sig *signal.Handler,
) *kubernetes.KubernetesBrowserManager {
	l := log.GetLogger().Named("k8s")
	bc, err := kubernetes.NewTemplatedBrowserConverter(
		cfg,
		templatesData[podTemplateFile],
		[]byte(templatesData[valuesFile]),
		l.Named("converter"),
	)
	if err != nil {
		InitLog.Fatalw("failed to initialize Browser to Pod converter", zap.Error(err))
	}

	watcher, err := kubernetes.NewPodWatcher(client, cfg.Lineage(), l.Named("watcher"))
	if err != nil {
		InitLog.Fatalw("failed to initialize Pod watcher", zap.Error(err))
	}
	sig.RegisterShutdownHook(watcher, watcher.Shutdown)

	backoff := wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2,
		Jitter:   0.3,
		Steps:    cfg.CreateRetries(),
		Cap:      cfg.CreateTimeout(),
	}

	return kubernetes.NewKubernetesBrowserManager(cat, client, bc, watcher, backoff, l.Named("manager"))
}
