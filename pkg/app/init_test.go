package app

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/config"
)

const (
	podData = "test_pod"
	valData = "test_values"
)

func Test_readKubeTemplates(t *testing.T) {
	InitLog = zaptest.NewLogger(t).Sugar()
	g := NewWithT(t)

	c := new(mocks.Config)
	dir := t.TempDir()
	setupTemplates(g, dir)

	c.EXPECT().KubeTemplatesPath().Return(dir).Once()
	got := readKubeTemplates(c)

	g.Expect(got).To(Equal(map[string]string{
		"pod-template.yaml": podData,
		"values.yaml":       valData,
	}))

	c.AssertExpectations(t)
}

const testBrowsersURL = "https://remote/config"

var brData = []byte("test_browsers")

func Test_loadBrowsersConfig_local(t *testing.T) {
	InitLog = zaptest.NewLogger(t).Sugar()
	g := NewWithT(t)

	c := new(mocks.Config)
	dir := t.TempDir()
	brFile := dir + "/brbr.txt"
	err := os.WriteFile(brFile, brData, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	c.EXPECT().BrowsersURI().Return([]string{brFile, testBrowsersURL}).Once()
	got := loadBrowsersConfig(c, nil)

	g.Expect(got).To(Equal(brData))
	c.AssertExpectations(t)
}

func Test_loadBrowsersConfig_FallbackRemote(t *testing.T) {
	InitLog = zaptest.NewLogger(t).Sugar()
	g := NewWithT(t)

	c := new(mocks.Config)
	hc := new(mocks.HTTPClient)

	c.EXPECT().BrowsersURI().Return([]string{"qqqqqq/bebebe", testBrowsersURL}).Once()
	hc.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		g.Expect(req.Method).To(Equal(http.MethodGet))
		g.Expect(req.URL.String()).To(Equal(testBrowsersURL))

		resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(brData))}
		return resp, nil
	}).Once()
	got := loadBrowsersConfig(c, hc)

	g.Expect(got).To(Equal(brData))
	c.AssertExpectations(t)
	hc.AssertExpectations(t)
}

func setupTemplates(g *WithT, dir string) {
	err := os.WriteFile(dir+"/pod-template.yaml", []byte(podData), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(dir+"/values.yaml", []byte(valData), 0644)
	g.Expect(err).ToNot(HaveOccurred())
}

func Test_detectBackend(t *testing.T) {
	tests := []struct {
		name         string
		cfgBackend   config.BackendType
		inKubernetes bool
		want         config.BackendType
	}{
		{
			name:         "kubernetes explicitly",
			cfgBackend:   config.BackendKubernetes,
			inKubernetes: false,
			want:         config.BackendKubernetes,
		},
		{
			name:         "auto in kubernetes",
			cfgBackend:   config.BackendAuto,
			inKubernetes: true,
			want:         config.BackendKubernetes,
		},
		{
			name:         "docker explicitly",
			cfgBackend:   config.BackendDocker,
			inKubernetes: true,
			want:         config.BackendDocker,
		},
		{
			name:         "docker auto",
			cfgBackend:   config.BackendAuto,
			inKubernetes: false,
			want:         config.BackendDocker,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			cfg := new(mocks.Config)
			InKubernetes = func() bool {
				return tt.inKubernetes
			}
			cfg.EXPECT().Backend().Return(tt.cfgBackend)
			got := detectBackend(cfg)
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_portMappingEnabled(t *testing.T) {
	tests := []struct {
		name              string
		dockerPortMapping config.PortMappingMode
		inDocker          bool
		want              bool
	}{
		{
			name:              "explicitly enabled",
			dockerPortMapping: config.PortMappingEnabled,
			inDocker:          true,
			want:              true,
		},
		{
			name:              "auto-disabled when in docker",
			dockerPortMapping: config.PortMappingAuto,
			inDocker:          true,
			want:              false,
		},
		{
			name:              "auto-enabled when not in docker",
			dockerPortMapping: config.PortMappingAuto,
			inDocker:          false,
			want:              true,
		},
		{
			name:              "explicitly disabled",
			dockerPortMapping: config.PortMappingDisabled,
			inDocker:          false,
			want:              false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			cfg := new(mocks.DockerConfig)
			cfg.EXPECT().DockerPortMapping().Return(tt.dockerPortMapping)
			InDocker = func() bool {
				return tt.inDocker
			}
			got := portMappingEnabled(cfg)
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_listen(t *testing.T) {
	tests := []struct {
		name         string
		listen       string
		inDocker     bool
		inKubernetes bool
		want         string
	}{
		{
			name:   "explicitly set",
			listen: "12345",
			want:   "12345",
		},
		{
			name:         "in kubernetes bind to all interfaces by default",
			inKubernetes: true,
			want:         "0.0.0.0:4444",
		},
		{
			name:     "in Docker bind to all interfaces by default",
			inDocker: true,
			want:     "0.0.0.0:4444",
		},
		{
			name: "locally bind to localhost by default",
			want: "127.0.0.1:4444",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			cfg := new(mocks.Config)
			cfg.EXPECT().Listen().Return(tt.listen).Once()
			InDocker = func() bool {
				return tt.inDocker
			}
			InKubernetes = func() bool {
				return tt.inKubernetes
			}
			got := listen(cfg)
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
