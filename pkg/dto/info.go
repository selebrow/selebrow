package dto

type AppInfo struct {
	Name   string `json:"name"`
	GitRef string `json:"gitRef"`
	GitSha string `json:"gitSha"`
}
