package models

import (
	"time"
)

type PWCapabilities struct {
	Platform         string
	Flavor           string
	Browser          string
	Version          string
	VNCEnabled       bool
	ScreenResolution string
	Env              []string
	Links            []string
	Hosts            []string
	Networks         []string
	Labels           map[string]string
}

func (caps *PWCapabilities) GetName() string {
	return caps.Browser
}

func (caps *PWCapabilities) GetVersion() string {
	return caps.Version
}

func (caps *PWCapabilities) GetPlatform() string {
	return caps.Platform
}

func (caps *PWCapabilities) GetResolution() string {
	return caps.ScreenResolution
}

func (caps *PWCapabilities) IsVNCEnabled() bool {
	return caps.VNCEnabled
}

func (caps *PWCapabilities) GetTestName() string {
	return ""
}

func (caps *PWCapabilities) GetEnvs() []string {
	return caps.Env
}

func (caps *PWCapabilities) GetTimeout() time.Duration {
	return 0
}

func (caps *PWCapabilities) GetRawCapabilities() []byte {
	return nil
}

func (caps *PWCapabilities) GetFlavor() string {
	return caps.Flavor
}

func (caps *PWCapabilities) GetLinks() []string {
	return caps.Links
}

func (caps *PWCapabilities) GetHosts() []string {
	return caps.Hosts
}

func (caps *PWCapabilities) GetNetworks() []string {
	return caps.Networks
}

func (caps *PWCapabilities) GetLabels() map[string]string {
	return caps.Labels
}
