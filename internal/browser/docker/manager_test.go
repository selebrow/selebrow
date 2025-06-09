package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/internal/browser/docker"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

const (
	testHost = "dockerhost"
	testNet  = "test-net"

	testBrowserProtocol models.BrowserProtocol = "test"

	testResolution  = "640x480x0"
	testContainerID = "abc321"
)

var (
	testImages = []string{"img1", "img2"}
	testError  = errors.New("testError")
	testIPs    = []string{"1.1.1.1", "4.3.2.1"}

	localIPsMock = func() ([]string, error) {
		return testIPs, nil
	}
)

type fakeNotFound struct{}

func (fakeNotFound) NotFound()     {}
func (fakeNotFound) Error() string { return "error fake not found" }

func TestKubernetesBrowserManager_NewDockerBrowserManager(t *testing.T) {
	tests := []struct {
		name       string
		opts       docker.DockerBrowserManagerOpts
		setupMocks func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient)
		wantErr    bool
	}{
		{
			name: "pullImages",
			opts: docker.DockerBrowserManagerOpts{
				Network:    testNet,
				PullImages: true,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().GetImages().Return(testImages).Once()
				client.EXPECT().ImageInspect(context.Background(), testImages[0]).Return(image.InspectResponse{}, nil).Once()
				client.EXPECT().ImageInspect(context.Background(), testImages[1]).Return(image.InspectResponse{}, &fakeNotFound{}).Once()
				client.EXPECT().ImagePull(context.Background(), testImages[1]).Return(nil).Once()
			},
		},
		{
			name: "pullImages inspect error",
			opts: docker.DockerBrowserManagerOpts{
				PullImages: true,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().GetImages().Return(testImages)
				client.EXPECT().ImageInspect(context.Background(), testImages[0]).Return(image.InspectResponse{}, testError).Once()
			},
			wantErr: true,
		},
		{
			name: "pullImages pull error",
			opts: docker.DockerBrowserManagerOpts{
				PullImages: true,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().GetImages().Return(testImages)
				client.EXPECT().ImageInspect(context.Background(), testImages[0]).Return(image.InspectResponse{}, &fakeNotFound{}).Once()
				client.EXPECT().ImagePull(context.Background(), testImages[0]).Return(testError).Once()
			},
			wantErr: true,
		},
		{
			name: "detectNetwork LocalIPs error",
			opts: docker.DockerBrowserManagerOpts{
				Network:    "",
				MapPorts:   false,
				PullImages: false,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				docker.LocalIPs = func() ([]string, error) {
					return nil, testError
				}
			},
			wantErr: true,
		},
		{
			name: "detectNetwork container list error",
			opts: docker.DockerBrowserManagerOpts{
				Network:    "",
				MapPorts:   false,
				PullImages: false,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				docker.LocalIPs = localIPsMock
				client.EXPECT().ContainerList(context.Background()).Return(nil, testError).Once()
			},
			wantErr: true,
		},
		{
			name: "detectNetwork no container matching IPs",
			opts: docker.DockerBrowserManagerOpts{
				Network:    "",
				MapPorts:   false,
				PullImages: false,
			},
			setupMocks: func(cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				docker.LocalIPs = localIPsMock
				client.EXPECT().ContainerList(context.Background()).Return([]container.Summary{}, nil).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			cat := new(mocks.BrowsersCatalog)
			client := new(mocks.DockerClient)

			tt.setupMocks(cat, client)
			m, err := docker.NewDockerBrowserManager(client, cat, tt.opts, zaptest.NewLogger(t))
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(m).ToNot(BeNil())
			}
			cat.AssertExpectations(t)
			client.AssertExpectations(t)
		})
	}
}

var (
	testEnv      = []string{"a=b", "b=c=d"}
	testLabels   = map[string]string{"k1": "v1", "k2": "v2"}
	testHosts    = []string{"aaa:1.2.3.4", "bbb:1.2.3.4"}
	testLinks    = []string{"cont1:domain.ltd"}
	testNetworks = []string{"net1"}

	testBrowsersConfig = models.BrowserImageConfig{
		Image: "apple/safari",
		Cmd:   []string{"run"},
		VersionTags: map[string]string{
			"135": "test-1",
		},
		Ports: map[models.ContainerPort]int{
			models.BrowserPort:   123,
			models.ClipboardPort: 777,
			models.VNCPort:       444,
		},
		Path: "/wd",
		Env:  map[string]string{"a": "b", "b": "c"},
		Limits: map[string]string{
			"cpu":    "1",
			"memory": "100500Gi",
		},
		Labels:  map[string]string{"main": "val", "k1": "ignore"},
		ShmSize: 10000,
		Tmpfs:   []string{"/tmp:opts", "/var/tmp"},
		Volumes: []string{"/src:/build"},
	}

	expConfig = &container.Config{
		Cmd:          []string{"run"},
		ExposedPorts: nat.PortSet{"123/tcp": struct{}{}, "777/tcp": struct{}{}},
		Env:          []string{"ENABLE_VNC=false", "SCREEN_RESOLUTION=640x480x0", "a=b", "b=c=d"},
		Image:        "apple/safari:test-1",
		Labels:       map[string]string{"main": "val", "k1": "v1", "k2": "v2"},
	}

	expPortBindings = nat.PortMap{
		"123/tcp": []nat.PortBinding{{HostIP: "", HostPort: ""}},
		"777/tcp": []nat.PortBinding{{HostIP: "", HostPort: ""}},
	}

	expNetworkingConfig = &network.NetworkingConfig{}

	createResp = container.CreateResponse{
		ID: testContainerID,
	}

	inspectRespNoPortMap = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase:    container.NetworkSettingsBase{},
			DefaultNetworkSettings: container.DefaultNetworkSettings{},
			Networks: map[string]*network.EndpointSettings{
				testNet: {
					IPAddress: "4.5.6.7",
				},
			},
		},
	}

	inspectRespPartialPortMap = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
					"777/tcp": []nat.PortBinding{},
				},
			},
			DefaultNetworkSettings: container.DefaultNetworkSettings{
				IPAddress: "4.5.6.7",
			},
		},
	}

	inspectRespPortMap = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
					"777/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "999",
						},
					},
				},
			},
			DefaultNetworkSettings: container.DefaultNetworkSettings{
				IPAddress: "4.5.6.7",
			},
		},
	}
)

func TestKubernetesBrowserManager_Allocate_PortMap_Mode(t *testing.T) {
	g := NewWithT(t)

	cat := new(mocks.BrowsersCatalog)
	client := new(mocks.DockerClient)

	client.EXPECT().GetHost().Return(testHost).Once()
	mgr, err := docker.NewDockerBrowserManager(client, cat, docker.DockerBrowserManagerOpts{
		Network:    testNet,
		MapPorts:   true,
		Privileged: true,
		PullImages: false,
	}, zaptest.NewLogger(t))
	g.Expect(err).ToNot(HaveOccurred())

	caps := createCaps("safari", "135", "def", false)

	cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()

	expHostConfig := getExpHostConfig(expPortBindings)
	client.EXPECT().ContainerCreate(context.TODO(), expConfig, expHostConfig, expNetworkingConfig, "").Return(createResp, nil).Once()
	client.EXPECT().ContainerStart(context.TODO(), testContainerID).Return(nil).Once()

	// test port mapping wait loop code path
	client.EXPECT().ContainerInspect(context.TODO(), testContainerID).Return(inspectRespNoPortMap, nil).Once()
	client.EXPECT().ContainerInspect(context.TODO(), testContainerID).Return(inspectRespPartialPortMap, nil).Once()

	client.EXPECT().ContainerInspect(context.TODO(), testContainerID).Return(inspectRespPortMap, nil).Once()
	wd, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())

	u := wd.GetURL()
	g.Expect(u.String()).To(Equal("http://dockerhost:787/wd"))
	g.Expect(wd.GetHost()).To(Equal("4.5.6.7:123"))

	g.Expect(wd.GetHostPort(models.ClipboardPort)).To(Equal("dockerhost:999"))
	g.Expect(wd.GetHostPort(models.VNCPort)).To(BeEmpty())

	client.EXPECT().ContainerRemove(context.TODO(), testContainerID, true).Return(nil).Once()
	wd.Close(context.TODO(), true)

	client.AssertExpectations(t)
}

var (
	testContainers = []container.Summary{
		{},
		{
			NetworkSettings: &container.NetworkSettingsSummary{},
		},
		{
			NetworkSettings: &container.NetworkSettingsSummary{
				Networks: map[string]*network.EndpointSettings{
					"some": {
						IPAddress: "1.2.3.4",
					},
					testNet: {
						IPAddress: "4.3.2.1",
					},
				},
			},
		},
	}
)

func TestKubernetesBrowserManager_Allocate_DirectConnection(t *testing.T) {
	g := NewWithT(t)

	cat := new(mocks.BrowsersCatalog)
	client := new(mocks.DockerClient)

	docker.LocalIPs = localIPsMock

	client.EXPECT().ContainerList(context.Background()).Return(testContainers, nil)
	mgr, err := docker.NewDockerBrowserManager(client, cat, docker.DockerBrowserManagerOpts{
		Network:    "", // should trigger network autodetect
		MapPorts:   false,
		Privileged: true,
		PullImages: false,
	}, zaptest.NewLogger(t))
	g.Expect(err).ToNot(HaveOccurred())

	caps := createCaps("safari", "135", "def", false)

	cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()

	expHostConfig := getExpHostConfig(nil)
	// check image pull code path
	client.EXPECT().
		ContainerCreate(context.TODO(), expConfig, expHostConfig, expNetworkingConfig, "").
		Return(container.CreateResponse{}, &fakeNotFound{}).
		Once()
	client.EXPECT().ImagePull(context.Background(), "apple/safari:test-1").Return(nil).Once()

	client.EXPECT().ContainerCreate(context.TODO(), expConfig, expHostConfig, expNetworkingConfig, "").Return(createResp, nil).Once()
	client.EXPECT().ContainerStart(context.TODO(), testContainerID).Return(nil).Once()
	client.EXPECT().ContainerInspect(context.TODO(), testContainerID).Return(inspectRespNoPortMap, nil).Once()
	wd, err := mgr.Allocate(context.TODO(), testBrowserProtocol, caps)
	g.Expect(err).ToNot(HaveOccurred())

	u := wd.GetURL()
	g.Expect(u.String()).To(Equal("http://4.5.6.7:123/wd"))
	g.Expect(wd.GetHost()).To(Equal("4.5.6.7:123"))

	g.Expect(wd.GetHostPort(models.ClipboardPort)).To(Equal("4.5.6.7:777"))
	g.Expect(wd.GetHostPort(models.VNCPort)).To(BeEmpty())

	client.EXPECT().ContainerRemove(context.TODO(), testContainerID, true).Return(nil).Once()
	wd.Close(context.TODO(), true)

	cat.AssertExpectations(t)
	client.AssertExpectations(t)
}

var (
	inspectRespNotRunning = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: false},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
				},
			},
		},
	}

	inspectRespNoNetwork = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
				},
			},
		},
	}

	inspectRespNoIP = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
				},
			},
			Networks: map[string]*network.EndpointSettings{
				testNet: {},
			},
		},
	}

	inspectRespNoMappedPort = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    testContainerID,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"123/tcp": []nat.PortBinding{
						{
							HostIP:   "0.0.0.0",
							HostPort: "787/tcp",
						},
					},
				},
			},
			Networks: map[string]*network.EndpointSettings{
				testNet: {
					IPAddress: "1.1.1.1",
				},
			},
		},
	}
)

func TestKubernetesBrowserManager_Allocate_Negative(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		setupMocks func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient)
	}{
		{
			name: "no image flavor available",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(models.BrowserImageConfig{}, false).Once()
			},
		},
		{
			name:    "create container no tag",
			version: "11111",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
			},
		},
		{
			name:    "create container error",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().
					ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").
					Return(container.CreateResponse{}, testError).
					Once()
			},
		},
		{
			name:    "create container pull error",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().
					ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").
					Return(container.CreateResponse{}, &fakeNotFound{}).
					Once()
				client.EXPECT().ImagePull(context.Background(), "apple/safari:test-1").Return(testError).Once()
			},
		},
		{
			name:    "create container pull cancel",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().
					ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").
					Return(container.CreateResponse{}, &fakeNotFound{}).
					Once()
				client.EXPECT().ImagePull(context.Background(), "apple/safari:test-1").
					RunAndReturn(func(ctx context.Context, _ string) error {
						cancel()
						time.Sleep(time.Second)   // simulate image pull in background
						return errors.New("test") // hack to avoid logger error
					}).Once()
			},
		},
		{
			name:    "start container error",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(testError).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
		{
			name:    "start container inspect error",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(nil).Once()
				client.EXPECT().ContainerInspect(ctx, testContainerID).Return(container.InspectResponse{}, testError).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
		{
			name:    "start container not running",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(nil).Once()
				client.EXPECT().ContainerInspect(ctx, testContainerID).Return(inspectRespNotRunning, nil).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
		{
			name:    "createBrowser no network",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(nil).Once()
				client.EXPECT().ContainerInspect(ctx, testContainerID).Return(inspectRespNoNetwork, nil).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
		{
			name:    "createBrowser no ip",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(nil).Once()
				client.EXPECT().ContainerInspect(ctx, testContainerID).Return(inspectRespNoIP, nil).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
		{
			name:    "createBrowser no mapped port",
			version: "135",
			setupMocks: func(ctx context.Context, cancel context.CancelFunc, cat *mocks.BrowsersCatalog, client *mocks.DockerClient) {
				cat.EXPECT().LookupBrowserImage(testBrowserProtocol, "safari", "def").Return(testBrowsersConfig, true).Once()
				client.EXPECT().ContainerCreate(ctx, mock.Anything, mock.Anything, mock.Anything, "").Return(createResp, nil).Once()
				client.EXPECT().ContainerStart(ctx, testContainerID).Return(nil).Once()
				client.EXPECT().ContainerInspect(ctx, testContainerID).Return(inspectRespNoMappedPort, nil).Once()
				client.EXPECT().ContainerRemove(context.Background(), testContainerID, true).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			cat := new(mocks.BrowsersCatalog)

			client := new(mocks.DockerClient)

			client.EXPECT().GetHost().Return(testHost).Once()
			mgr, err := docker.NewDockerBrowserManager(client, cat, docker.DockerBrowserManagerOpts{
				Network:    testNet,
				MapPorts:   true,
				Privileged: true,
				PullImages: false,
			}, zaptest.NewLogger(t))
			g.Expect(err).ToNot(HaveOccurred())

			caps := createCaps("safari", tt.version, "def", false)

			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()
			tt.setupMocks(ctx, cancel, cat, client)
			_, err = mgr.Allocate(ctx, testBrowserProtocol, caps)
			g.Expect(err).To(HaveOccurred())

			cat.AssertExpectations(t)
			client.AssertExpectations(t)
		})
	}
}

func createCaps(name, version, flavor string, vncEnabled bool) *mocks.Capabilities {
	caps := new(mocks.Capabilities)
	caps.EXPECT().GetName().Return(name)
	caps.EXPECT().GetVersion().Return(version)
	caps.EXPECT().IsVNCEnabled().Return(vncEnabled)
	caps.EXPECT().GetFlavor().Return(flavor)
	caps.EXPECT().GetEnvs().Return(testEnv)
	caps.EXPECT().GetResolution().Return(testResolution)
	caps.EXPECT().GetLabels().Return(testLabels)
	caps.EXPECT().GetHosts().Return(testHosts)
	caps.EXPECT().GetLinks().Return(testLinks)
	caps.EXPECT().GetNetworks().Return(testNetworks)
	return caps
}

func getExpHostConfig(portBindings nat.PortMap) *container.HostConfig {
	return &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyDisabled},
		Binds:         []string{"/src:/build"},
		NetworkMode:   "test-net",
		PortBindings:  portBindings,
		ExtraHosts:    []string{"aaa:1.2.3.4", "bbb:1.2.3.4"},
		Links:         []string{"cont1:domain.ltd"},
		Privileged:    true,
		ShmSize:       10000,
		Tmpfs: map[string]string{
			"/tmp":     "opts",
			"/var/tmp": "",
		},
		Resources: container.Resources{
			Memory:   107911053312000,
			NanoCPUs: 1000000000,
		},
	}
}
