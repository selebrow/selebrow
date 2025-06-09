package config

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "positive 1",
			args: []string{"--backend", "kubernetes", "--docker-port-mapping", "disabled"},
		},
		{
			name: "positive 2",
			args: []string{"--backend", "docker", "--docker-port-mapping", "enabled"},
		},
		{
			name: "positive auto",
			args: []string{"--backend", "auto", "--docker-port-mapping", "auto"},
		},
		{
			name:    "incorrect backend",
			args:    []string{"--backend", "qwe", "--docker-port-mapping", "enabled"},
			wantErr: true,
		},
		{
			name:    "incorrect docker port mapping",
			args:    []string{"--backend", "docker", "--docker-port-mapping", "qwe"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			f := pflag.NewFlagSet("test", pflag.ContinueOnError)
			f.String(backend, "", "")
			f.String(dockerPortMapping, "", "")

			err := f.Parse(tt.args)
			g.Expect(err).ToNot(HaveOccurred())

			got, err := NewConfig(viper.New(), f)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(got).ToNot(BeNil())
			}
		})
	}
}

func TestConfigViper(t *testing.T) {
	g := NewWithT(t)
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f.String("listen", "", "")
	f.String("namespace", "def", "")
	f.Int("pool-max-idle", 5, "")
	f.Bool("ui", true, "")
	err := f.Parse([]string{"--listen=:1234"})
	g.Expect(err).ToNot(HaveOccurred())

	genLineage = func() string {
		return "155"
	}

	v := viper.New()
	v.Set("browsers-uri", "file.txt")
	v.Set("fallback-browsers-uri", "http://test")
	v.Set("backend", "auto")
	v.Set("cluster-mode-out", true)
	v.Set("pool-max-age", 2*time.Minute)
	v.Set("pool-idle-timeout", 23*time.Second)
	v.Set("create-timeout", 3*time.Minute)
	v.Set("connect-timeout", 5*time.Minute)
	v.Set("kube-config", "/asd")

	v.Set("quota-limit", "123")
	v.Set("queue-size", 13)
	v.Set("queue-timeout", "1h")

	v.Set("create-retries", "44")

	v.Set("docker-network", "test-net")
	v.Set("docker-privileged", "true")
	v.Set("docker-port-mapping", "enabled")
	v.Set("docker-pull-images", "true")
	v.Set("docker-platform", "cp/m")

	v.Set("vnc-password", "12345")

	t.Setenv("CI_JOB_ID", "321")
	t.Setenv("CI_PROJECT_NAMESPACE", "test")
	t.Setenv("CI_PROJECT_NAME", "test-proj")
	t.Setenv("NAMESPACE", "test-ns")
	t.Setenv("SB_CONFIG_NAME", "config")
	t.Setenv("SB_POOL_MAX_IDLE", "55")
	t.Setenv("SB_KUBE_TEMPLATES_PATH", "qqq/")

	cfg, err := NewConfig(v, f)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(cfg.Listen()).To(Equal(":1234"))
	g.Expect(cfg.Backend()).To(Equal(BackendAuto))
	g.Expect(cfg.BrowsersURI()).To(Equal([]string{"file.txt", "http://test"}))
	g.Expect(cfg.Lineage()).To(Equal("155"))
	g.Expect(cfg.JobID()).To(Equal("321"))
	g.Expect(cfg.ProjectNamespace()).To(Equal("test"))
	g.Expect(cfg.ProjectName()).To(Equal("test-proj"))
	g.Expect(cfg.Namespace()).To(Equal("test-ns"))
	g.Expect(cfg.KubeClusterModeOut()).To(BeFalse())
	g.Expect(cfg.MaxIdle()).To(Equal(55))
	g.Expect(cfg.ProxyDelete()).To(BeTrue())
	g.Expect(cfg.MaxAge()).To(Equal(2 * time.Minute))
	g.Expect(cfg.IdleTimeout()).To(Equal(23 * time.Second))
	g.Expect(cfg.CreateTimeout()).To(Equal(3 * time.Minute))
	g.Expect(cfg.ConnectTimeout()).To(Equal(5 * time.Minute))
	g.Expect(cfg.KubeConfig()).To(Equal("/asd"))
	g.Expect(cfg.KubeTemplatesPath()).To(Equal("qqq/"))

	g.Expect(cfg.QuotaLimit()).To(Equal(123))
	g.Expect(cfg.QueueSize()).To(Equal(13))
	g.Expect(cfg.QueueTimeout()).To(Equal(time.Hour))

	g.Expect(cfg.CreateRetries()).To(Equal(44))

	g.Expect(cfg.DockerPortMapping()).To(Equal(PortMappingEnabled))
	g.Expect(cfg.DockerNetwork()).To(Equal("test-net"))
	g.Expect(cfg.DockerPrivileged()).To(BeTrue())
	g.Expect(cfg.DockerPullImages()).To(BeTrue())
	g.Expect(cfg.DockerPlatform()).To(Equal("cp/m"))

	g.Expect(cfg.UI()).To(BeTrue())
	g.Expect(cfg.VNCPassword()).To(Equal("12345"))
}
