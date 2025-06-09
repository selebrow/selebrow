package dto

type Config struct {
	Files map[string]ConfigFile `json:"files"`
}

type ConfigFile struct {
	SHA256Sum string `json:"sha256Sum"`
}
