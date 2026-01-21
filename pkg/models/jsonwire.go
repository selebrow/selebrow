package models

// JsonWireCapabilities JsonWire capabilities model
// full description at https://www.selenium.dev/documentation/legacy/json_wire_protocol/
type JsonWireCapabilities struct {
	DesiredCapabilities map[string]interface{} `json:"desiredCapabilities,omitempty"`
}

func (caps *JsonWireCapabilities) UpdateProxy(proxy *ProxyOptions) {
	caps.DesiredCapabilities["proxy"] = proxy
}
