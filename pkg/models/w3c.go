package models

import (
	"time"

	"dario.cat/mergo"
)

// W3CCapabilities partial WebDriver capabilities specs model
// see details at https://www.w3.org/TR/webdriver/#capabilities
type W3CCapabilities struct {
	AlwaysMatch     *W3CBase  `json:"alwaysMatch,omitempty"`
	FirstMatch      []W3CBase `json:"firstMatch,omitempty"`
	Merged          W3CBase   `json:"-"`
	RawCapabilities []byte    `json:"-"`
}

func (caps *W3CCapabilities) GetRawCapabilities() []byte {
	return caps.RawCapabilities
}

type W3CBase struct {
	Name            string           `json:"browserName,omitempty"`
	DeviceName      string           `json:"deviceName,omitempty"`
	Version         string           `json:"browserVersion,omitempty"`
	Platform        string           `json:"platformName,omitempty"`
	SelenoidOptions *SelenoidOptions `json:"selenoid:options,omitempty"`
}

func (caps *W3CCapabilities) Merge() {
	if caps.AlwaysMatch != nil {
		_ = mergo.Merge(&caps.Merged, caps.AlwaysMatch)
	}
	if len(caps.FirstMatch) > 0 {
		for _, c := range caps.FirstMatch {
			_ = mergo.Merge(&caps.Merged, c)
		}
	}
}

func (caps *W3CCapabilities) GetName() string {
	if caps.Merged.Name != "" {
		return caps.Merged.Name
	}
	return caps.Merged.DeviceName
}

func (caps *W3CCapabilities) GetVersion() string {
	return caps.Merged.Version
}

func (caps *W3CCapabilities) GetPlatform() string {
	return caps.Merged.Platform
}

func (caps *W3CCapabilities) GetResolution() string {
	if caps.Merged.SelenoidOptions == nil {
		return ""
	}
	return caps.Merged.SelenoidOptions.ScreenResolution
}

func (caps *W3CCapabilities) IsVNCEnabled() bool {
	if caps.Merged.SelenoidOptions == nil {
		return false
	}
	return caps.Merged.SelenoidOptions.EnableVNC
}

func (caps *W3CCapabilities) GetTestName() string {
	if caps.Merged.SelenoidOptions == nil {
		return ""
	}
	return caps.Merged.SelenoidOptions.TestName
}

func (caps *W3CCapabilities) GetEnvs() []string {
	if caps.Merged.SelenoidOptions == nil {
		return []string{}
	}
	return caps.Merged.SelenoidOptions.Env
}

func (caps *W3CCapabilities) GetTimeout() time.Duration {
	if caps.Merged.SelenoidOptions == nil {
		return 0
	}
	return time.Duration(caps.Merged.SelenoidOptions.SessionTimeout)
}

func (caps *W3CCapabilities) GetFlavor() string {
	if caps.Merged.SelenoidOptions == nil {
		return ""
	}
	return caps.Merged.SelenoidOptions.Flavor
}

func (caps *W3CCapabilities) GetLinks() []string {
	if caps.Merged.SelenoidOptions == nil {
		return nil
	}
	return caps.Merged.SelenoidOptions.Links
}

func (caps *W3CCapabilities) GetHosts() []string {
	if caps.Merged.SelenoidOptions == nil {
		return nil
	}
	return caps.Merged.SelenoidOptions.Hosts
}

func (caps *W3CCapabilities) GetNetworks() []string {
	if caps.Merged.SelenoidOptions == nil {
		return nil
	}
	return caps.Merged.SelenoidOptions.Networks
}

func (caps *W3CCapabilities) GetLabels() map[string]string {
	if caps.Merged.SelenoidOptions == nil {
		return nil
	}
	return caps.Merged.SelenoidOptions.Labels
}
