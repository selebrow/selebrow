package browsers

import (
	"fmt"
	"slices"

	"github.com/distribution/reference"
	"gopkg.in/yaml.v3"

	"github.com/selebrow/selebrow/pkg/browser"
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/models"
)

const (
	defaultFlavor = "default"
)

type BrowsersCatalog interface {
	LookupBrowserImage(protocol models.BrowserProtocol, name, flavor string) (models.BrowserImageConfig, bool)
	GetBrowsers(protocol models.BrowserProtocol, flavor string) (result []dto.Browser)
	GetImages() (result []string)
}

type YamlBrowsersCatalog struct {
	cat models.BrowserCatalog
}

func (b *YamlBrowsersCatalog) GetImages() (result []string) {
	for _, browsers := range b.cat {
		for _, browser := range browsers {
			for _, image := range browser.Images {
				for _, tag := range image.VersionTags {
					result = append(result, fmt.Sprintf("%s:%s", image.Image, tag))
				}
			}
		}
	}
	slices.Sort(result)
	result = slices.Compact(result)
	return
}

func NewYamlBrowsersCatalog(data []byte, imageRegistry string) (*YamlBrowsersCatalog, error) {
	cat := make(models.BrowserCatalog)
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	if err := setImageRegistry(cat, imageRegistry); err != nil {
		return nil, err
	}
	// TODO validate
	return &YamlBrowsersCatalog{cat: cat}, nil
}

func (b *YamlBrowsersCatalog) LookupBrowserImage(protocol models.BrowserProtocol, name, flavor string) (models.BrowserImageConfig, bool) {
	browsers, ok := b.cat[protocol]
	if !ok {
		return models.BrowserImageConfig{}, ok
	}

	cfg, ok := browsers[name]
	if !ok {
		return models.BrowserImageConfig{}, ok
	}

	if flavor == "" {
		flavor = defaultFlavor
	}
	ic, ok := cfg.Images[flavor]
	if !ok {
		return models.BrowserImageConfig{}, false
	}
	return *ic, ok
}

func (b *YamlBrowsersCatalog) GetBrowsers(
	protocol models.BrowserProtocol,
	flavor string,
) []dto.Browser {
	browsers, ok := b.cat[protocol]
	if !ok {
		return nil
	}

	if flavor == "" {
		flavor = defaultFlavor
	}

	result := []dto.Browser{}
	for name, cfg := range browsers {
		ic, ok := cfg.Images[flavor]
		if ok {
			var bv []dto.BrowserVersion
			for ver := range ic.VersionTags {
				bv = append(bv, dto.BrowserVersion{
					Number:   ver,
					Platform: browser.DefaultPlatform,
				})
			}
			result = append(result, dto.Browser{
				Name:            name,
				DefaultVersion:  ic.DefaultVersion,
				DefaultPlatform: browser.DefaultPlatform,
				Versions:        bv,
			})
		}
	}

	return result
}

func setImageRegistry(cat models.BrowserCatalog, newRegistry string) error {
	if newRegistry == "" {
		return nil
	}
	for _, browsers := range cat {
		for _, browser := range browsers {
			for _, image := range browser.Images {
				newImage, err := replaceRegistry(image.Image, newRegistry)
				if err != nil {
					return err
				}
				image.Image = newImage
			}
		}
	}
	return nil
}

func replaceRegistry(image, newRegistry string) (string, error) {
	// Parse and normalize (so "nginx" becomes "docker.io/library/nginx", etc.)
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", err
	}

	// Extract original components
	path := reference.Path(named) // repository path without domain

	// Build a new base name with the new domain
	base := newRegistry + "/" + path
	newNamed, err := reference.ParseNormalizedNamed(base)
	if err != nil {
		return "", err
	}

	return newNamed.String(), nil
}
