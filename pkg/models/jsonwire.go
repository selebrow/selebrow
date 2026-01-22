package models

import "strings"

// JsonWireCapabilities JsonWire capabilities model
// full description at https://www.selenium.dev/documentation/legacy/json_wire_protocol/
type JsonWireCapabilities struct {
	DesiredCapabilities map[string]interface{} `json:"desiredCapabilities,omitempty"`
}

func (caps *JsonWireCapabilities) UpdateProxy(proxy *ProxyOptions) {
	p := *proxy
	if proxy.NoProxy != nil {
		//nolint:errcheck // panic can't happen (or let it fail)
		proxyArray := proxy.NoProxy.([]string)
		p.NoProxy = strings.Join(proxyArray, ",")
	}
	caps.DesiredCapabilities["proxy"] = &p
}
