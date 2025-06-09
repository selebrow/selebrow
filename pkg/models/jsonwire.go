package models

import (
	"time"
)

// JsonWireCapabilities Partial JsonWire specs model
// full description at https://www.selenium.dev/documentation/legacy/desired_capabilities/
type JsonWireCapabilities struct {
	Name            string           `json:"browserName,omitempty"`
	DeviceName      string           `json:"deviceName,omitempty"`
	Version         string           `json:"version,omitempty"`
	Platform        string           `json:"platform,omitempty"`
	SelenoidOptions *SelenoidOptions `json:"selenoid:options,omitempty"`
	RawCapabilities []byte           `json:"-"`
}

func (caps *JsonWireCapabilities) GetRawCapabilities() []byte {
	return caps.RawCapabilities
}

func (caps *JsonWireCapabilities) GetName() string {
	if caps.DeviceName != "" {
		return caps.DeviceName
	}
	return caps.Name
}

func (caps *JsonWireCapabilities) GetVersion() string {
	return caps.Version
}

func (caps *JsonWireCapabilities) GetPlatform() string {
	return caps.Platform
}

func (caps *JsonWireCapabilities) GetResolution() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.ScreenResolution
}

func (caps *JsonWireCapabilities) IsVNCEnabled() bool {
	if caps.SelenoidOptions == nil {
		return false
	}
	return caps.SelenoidOptions.EnableVNC
}

func (caps *JsonWireCapabilities) GetTestName() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.TestName
}

func (caps *JsonWireCapabilities) GetEnvs() []string {
	if caps.SelenoidOptions == nil {
		return []string{}
	}
	return caps.SelenoidOptions.Env
}

func (caps *JsonWireCapabilities) GetTimeout() time.Duration {
	if caps.SelenoidOptions == nil {
		return 0
	}
	return time.Duration(caps.SelenoidOptions.SessionTimeout)
}

func (caps *JsonWireCapabilities) GetFlavor() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.Flavor
}

func (caps *JsonWireCapabilities) GetLinks() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Links
}

func (caps *JsonWireCapabilities) GetHosts() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Hosts
}

func (caps *JsonWireCapabilities) GetNetworks() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Networks
}

func (caps *JsonWireCapabilities) GetLabels() map[string]string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Labels
}
