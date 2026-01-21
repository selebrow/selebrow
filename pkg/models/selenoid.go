package models

//nolint:lll  // many tags
type SelenoidOptions struct {
	TestName         string            `json:"name,omitempty"                  jsonwire:"name,omitempty"                  w3c:"name,omitempty"`
	SessionTimeout   Duration          `json:"sessionTimeout,omitempty"        jsonwire:"sessionTimeout,omitempty"        w3c:"sessionTimeout,omitempty"`
	ScreenResolution string            `json:"screenResolution,omitempty"      jsonwire:"screenResolution,omitempty"      w3c:"screenResolution,omitempty"`
	EnableVNC        bool              `json:"enableVNC,omitempty"             jsonwire:"enableVNC,omitempty"             w3c:"enableVNC,omitempty"`
	Env              []string          `json:"env,omitempty"                   jsonwire:"env,omitempty"                   w3c:"env,omitempty"`
	Flavor           string            `json:"flavor,omitempty"                jsonwire:"flavor,omitempty"                w3c:"flavor,omitempty"`
	Links            []string          `json:"applicationContainers,omitempty" jsonwire:"applicationContainers,omitempty" w3c:"applicationContainers,omitempty"`
	Hosts            []string          `json:"hostsEntries,omitempty"          jsonwire:"hostsEntries,omitempty"          w3c:"hostsEntries,omitempty"`
	Networks         []string          `json:"additionalNetworks,omitempty"    jsonwire:"additionalNetworks,omitempty"    w3c:"additionalNetworks,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"                jsonwire:"labels,omitempty"                w3c:"labels,omitempty"`
}
