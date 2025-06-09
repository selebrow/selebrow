package models

type SelenoidOptions struct {
	TestName         string            `json:"name,omitempty"`
	SessionTimeout   Duration          `json:"sessionTimeout,omitempty"`
	ScreenResolution string            `json:"screenResolution,omitempty"`
	EnableVNC        bool              `json:"enableVNC,omitempty"`
	Env              []string          `json:"env,omitempty"`
	Flavor           string            `json:"flavor,omitempty"`
	Links            []string          `json:"applicationContainers,omitempty"`
	Hosts            []string          `json:"hostsEntries,omitempty"`
	Networks         []string          `json:"additionalNetworks,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}
