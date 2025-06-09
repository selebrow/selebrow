package kubeapi

import "os"

func InKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}
