package dto

type Browser struct {
	Name            string
	DefaultVersion  string
	DefaultPlatform string
	Versions        []BrowserVersion
}

type BrowserVersion struct {
	Number   string
	Platform string
}
