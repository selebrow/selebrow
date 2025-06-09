package models

type BrowserProtocol string

const (
	WebdriverProtocol  BrowserProtocol = "webdriver"
	PlaywrightProtocol BrowserProtocol = "playwright"
)

type ContainerPort string

const (
	VNCPort        ContainerPort = "vnc"
	DevtoolsPort   ContainerPort = "devtools"
	FileserverPort ContainerPort = "fileserver"
	ClipboardPort  ContainerPort = "clipboard"
	BrowserPort    ContainerPort = "browser"
)

type BrowserCatalog map[BrowserProtocol]Browsers
type Browsers map[string]BrowserConfig

type BrowserConfig struct {
	Images map[string]BrowserImageConfig `yaml:"images"`
}

type BrowserImageConfig struct {
	Image          string                `yaml:"image"`
	Cmd            []string              `yaml:"cmd"`
	DefaultVersion string                `yaml:"defaultVersion"`
	VersionTags    map[string]string     `yaml:"versionTags"`
	Ports          map[ContainerPort]int `yaml:"ports"`
	Path           string                `yaml:"path"`
	Env            map[string]string     `yaml:"env"`
	Limits         map[string]string     `yaml:"limits"`
	Labels         map[string]string     `yaml:"labels"`
	ShmSize        int64                 `yaml:"shmSize"`
	Tmpfs          []string              `yaml:"tmpfs"`
	Volumes        []string              `yaml:"volumes"`
}

func (c BrowserImageConfig) GetTag(version string) (string, bool) {
	if version == "" {
		version = c.DefaultVersion
	}
	t, ok := c.VersionTags[version]
	if !ok {
		return "", false
	}
	return t, true
}

func (c BrowserImageConfig) GetPorts(vncEnabled bool) map[ContainerPort]int {
	res := make(map[ContainerPort]int)
	for k, v := range c.Ports {
		if !vncEnabled && k == VNCPort {
			continue
		}
		res[k] = v
	}
	return res
}
