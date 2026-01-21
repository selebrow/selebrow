package models

// W3CCapabilities WebDriver capabilities model
// see details at https://www.w3.org/TR/webdriver2/#capabilities
type W3CCapabilities struct {
	AlwaysMatch map[string]interface{}   `json:"alwaysMatch,omitempty"`
	FirstMatch  []map[string]interface{} `json:"firstMatch,omitempty"`
}

func (caps *W3CCapabilities) Merge() map[string]interface{} {
	merged := make(map[string]interface{})
	if caps.AlwaysMatch != nil {
		merged = deepMergeMaps(merged, caps.AlwaysMatch)
	}
	for _, c := range caps.FirstMatch {
		merged = deepMergeMaps(merged, c)
	}
	return merged
}

func (caps *W3CCapabilities) UpdateProxy(proxy *ProxyOptions) {
	if caps.AlwaysMatch == nil {
		caps.AlwaysMatch = make(map[string]interface{})
	}
	caps.AlwaysMatch["proxy"] = proxy
}

func deepMergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		if vSrc, ok := v.(map[string]interface{}); ok {
			if vDst, ok := dst[k].(map[string]interface{}); ok {
				// Key exists in both and both values are maps, so recurse
				dst[k] = deepMergeMaps(vDst, vSrc)
				continue
			}
		}
		// add the value from src if it doesn't exist in dst
		if _, ok := dst[k]; !ok {
			dst[k] = v
		}
	}
	return dst
}
