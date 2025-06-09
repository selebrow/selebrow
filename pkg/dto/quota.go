package dto

type QuotaUsage struct {
	Limit     int `json:"limit"`
	Allocated int `json:"allocated"`
}
