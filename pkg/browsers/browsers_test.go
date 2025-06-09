package browsers

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/models"
)

const data1 = `
webdriver:
  chrome:    
    images:
      default:
        image: repo.tld/webdriver/chrome
        defaultVersion: "116.0"
        versionTags:
          "116.0": chrome_116.0
        ports: 
          browser: 4444
          vnc: 5900
        path: /wd/hub
        env:
          K1: V1
        limits:
          cpu: 1
          memory: 2Gi
      cp: 
        image: inner-repo.tld/selebrow/chrome-cp
        defaultVersion: "115.0"
        versionTags:
          "116.0": release-116.0
          "115.0": release-115.0
        ports: 
          browser: 4444
        path: /wd/hub
        env:
          K2: V2
        limits:
          cpu: 1
          memory: 2Gi
  firefox:
    images:
      default:
        image: repo.tld/webdriver/firefox
        defaultVersion: "100.0"
        versionTags:
          "100.0": firefox_100.0
        ports: 
          browser: 4444
          vnc: 5900
        path: /
        env:
          K3: V3
        limits:
          cpu: 1
          memory: 2Gi
playwright:
  webkit:
    images:
      default:
        image: repo.tld/playwright/webkit
        defaultVersion: "16.0"
        versionTags:
          "16.0": "1.23.1"
        ports: 
          browser: 8000
        path: /
        env:
          K4: V4
        limits:
          cpu: 1
          memory: 4Gi
`

func TestBrowsersCatalog_GetBrowsers(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(data1))
	g.Expect(err).ToNot(HaveOccurred())

	got := cat.GetBrowsers(models.PlaywrightProtocol, "")

	exp := []dto.Browser{
		{
			Name:            "webkit",
			DefaultVersion:  "16.0",
			DefaultPlatform: browser.DefaultPlatform,
			Versions: []dto.BrowserVersion{
				{
					Number:   "16.0",
					Platform: browser.DefaultPlatform,
				},
			},
		},
	}
	g.Expect(got).To(ConsistOf(exp))
}

func TestBrowsersCatalog_GetImages(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(data1))
	g.Expect(err).ToNot(HaveOccurred())

	got := cat.GetImages()

	g.Expect(got).To(ConsistOf([]string{
		"inner-repo.tld/selebrow/chrome-cp:release-115.0",
		"inner-repo.tld/selebrow/chrome-cp:release-116.0",
		"repo.tld/playwright/webkit:1.23.1",
		"repo.tld/webdriver/chrome:chrome_116.0",
		"repo.tld/webdriver/firefox:firefox_100.0",
	}))
}

func TestBrowsersCatalog_GetBrowsers_Default_Flavor(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(data1))
	g.Expect(err).ToNot(HaveOccurred())

	got := cat.GetBrowsers(models.PlaywrightProtocol, "some")

	g.Expect(got).To(Equal([]dto.Browser{}))
}

func TestBrowsersCatalog_GetBrowsers_Not_Configured(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(``))
	g.Expect(err).ToNot(HaveOccurred())

	got := cat.GetBrowsers(models.PlaywrightProtocol, "")

	g.Expect(got).To(BeNil())
}

func TestBrowsersCatalog_LookupBrowserImage(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(data1))
	g.Expect(err).ToNot(HaveOccurred())

	_, found := cat.LookupBrowserImage(models.WebdriverProtocol, "safari", "")
	g.Expect(found).To(BeFalse())

	_, found = cat.LookupBrowserImage(models.WebdriverProtocol, "chrome", "performance")
	g.Expect(found).To(BeFalse())

	ver, found := cat.LookupBrowserImage(models.WebdriverProtocol, "chrome", "default")
	g.Expect(found).To(BeTrue())

	exp := models.BrowserImageConfig{
		Image:          "repo.tld/webdriver/chrome",
		DefaultVersion: "116.0",
		VersionTags: map[string]string{
			"116.0": "chrome_116.0",
		},
		Path: "/wd/hub",
		Ports: map[models.ContainerPort]int{
			models.BrowserPort: 4444,
			models.VNCPort:     5900,
		},
		Env: map[string]string{
			"K1": "V1",
		},
		Limits: map[string]string{
			"cpu":    "1",
			"memory": "2Gi",
		},
	}
	g.Expect(ver).To(Equal(exp))

	ver, found = cat.LookupBrowserImage(models.WebdriverProtocol, "chrome", "")
	g.Expect(found).To(BeTrue())
	g.Expect(ver).To(Equal(exp))
}

func TestBrowsersCatalog_LookupBrowserImage_Not_Configured(t *testing.T) {
	g := NewWithT(t)

	cat, err := NewYamlBrowsersCatalog([]byte(``))
	g.Expect(err).ToNot(HaveOccurred())

	_, found := cat.LookupBrowserImage(models.WebdriverProtocol, "safari", "")
	g.Expect(found).To(BeFalse())
}

func TestNewYamlBrowsersCatalog_Negative(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "malformed yaml",
			data: "qqqq",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			_, err := NewYamlBrowsersCatalog([]byte(tt.data))
			g.Expect(err).To(HaveOccurred())
		})
	}
}
