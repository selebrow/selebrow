package models

import "time"

// Capabilities Meaningful capabilities structure
type Capabilities struct {
	Name            string           `jsonwire:"browserName,omitempty"      w3c:"browserName,omitempty"`
	DeviceName      string           `jsonwire:"deviceName,omitempty"       w3c:"deviceName,omitempty"`
	Version         string           `jsonwire:"version,omitempty"          w3c:"browserVersion,omitempty"`
	Platform        string           `jsonwire:"platform,omitempty"         w3c:"platformName,omitempty"`
	Proxy           *ProxyOptions    `jsonwire:"proxy,omitempty"            w3c:"proxy,omitempty"`
	SelenoidOptions *SelenoidOptions `jsonwire:"selenoid:options,omitempty" w3c:"selenoid:options,omitempty"`
	RawCapabilities []byte           `jsonwire:"-"                          w3c:"-"`
}

func (caps *Capabilities) GetRawCapabilities() []byte {
	return caps.RawCapabilities
}

func (caps *Capabilities) GetName() string {
	if caps.DeviceName != "" {
		return caps.DeviceName
	}
	return caps.Name
}

func (caps *Capabilities) GetVersion() string {
	return caps.Version
}

func (caps *Capabilities) GetPlatform() string {
	return caps.Platform
}

func (caps *Capabilities) GetResolution() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.ScreenResolution
}

func (caps *Capabilities) IsVNCEnabled() bool {
	if caps.SelenoidOptions == nil {
		return false
	}
	return caps.SelenoidOptions.EnableVNC
}

func (caps *Capabilities) GetTestName() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.TestName
}

func (caps *Capabilities) GetEnvs() []string {
	if caps.SelenoidOptions == nil {
		return []string{}
	}
	return caps.SelenoidOptions.Env
}

func (caps *Capabilities) GetTimeout() time.Duration {
	if caps.SelenoidOptions == nil {
		return 0
	}
	return caps.SelenoidOptions.SessionTimeout.Duration
}

func (caps *Capabilities) GetFlavor() string {
	if caps.SelenoidOptions == nil {
		return ""
	}
	return caps.SelenoidOptions.Flavor
}

func (caps *Capabilities) GetLinks() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Links
}

func (caps *Capabilities) GetHosts() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Hosts
}

func (caps *Capabilities) GetNetworks() []string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Networks
}

func (caps *Capabilities) GetLabels() map[string]string {
	if caps.SelenoidOptions == nil {
		return nil
	}
	return caps.SelenoidOptions.Labels
}
