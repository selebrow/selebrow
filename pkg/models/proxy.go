package models

import "strings"

// Partial proxy configuration capability model (see https://	www.w3.org/TR/webdriver2/#proxy)
type (
	ProxyType    string
	ProxyOptions struct {
		ProxyType ProxyType `json:"proxyType,omitempty" jsonwire:"proxyType,omitempty" w3c:"proxyType,omitempty"`
		HTTPProxy string    `json:"httpProxy,omitempty" jsonwire:"httpProxy,omitempty" w3c:"httpProxy,omitempty"`
		SSLProxy  string    `json:"sslProxy,omitempty"  jsonwire:"sslProxy,omitempty"  w3c:"sslProxy,omitempty"`
		// JsonWire - string, W3C WebDriver - array of strings
		NoProxy any `json:"noProxy,omitempty"   jsonwire:"noProxy,omitempty"   w3c:"noProxy,omitempty"`
	}
)

const (
	ProxyTypeDirect     ProxyType = "direct"
	ProxyTypeManual     ProxyType = "manual"
	ProxyTypePAC        ProxyType = "pac"
	ProxyTypeSystem     ProxyType = "system"
	ProxyTypeAutoDetect ProxyType = "autodetect"
)

func NewHTTPProxy(proxyHost, noProxyStr string) *ProxyOptions {
	noProxy := strings.Split(noProxyStr, ",")
	for i, p := range noProxy {
		noProxy[i] = strings.TrimSpace(p)
	}
	return &ProxyOptions{
		ProxyType: ProxyTypeManual,
		HTTPProxy: proxyHost,
		SSLProxy:  proxyHost,
		NoProxy:   noProxy,
	}
}
