package dto

type Status struct {
	Total    int                        `json:"total"`
	Sessions map[string][]SessionStatus `json:"sessions"`
}

type SessionStatus struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
