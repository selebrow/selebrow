package models

type WebDriverStatus struct {
	Value WebDriverReadyStatus `json:"value"`
}

type WebDriverReadyStatus struct {
	Ready bool `json:"ready"`
}

func NewWebDriverStatus(ready bool) *WebDriverStatus {
	return &WebDriverStatus{
		Value: WebDriverReadyStatus{
			Ready: ready,
		},
	}
}
