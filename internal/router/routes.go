package router

import "fmt"

const (
	WDHUBPath    = "/wd/hub"
	SessionPath  = "/session"
	SessionParam = "sess"

	PWPath       = "/pw"
	NameParam    = "name"
	VersionParam = "version"
	FlavorQParam = "flavor"
	ProtoQParam  = "protocol"

	VNCPath = "/vnc"

	UIRoot   = "/ui"
	UIWDRoot = "/wd"
	UIPWRoot = "/pw"

	UIVNCPath   = "/vnc"
	UIResetPath = "/reset"
)

func SessRoute(s string) string {
	return fmt.Sprintf(s, SessionParam)
}

func NameRoute(s string) string {
	return fmt.Sprintf(s, NameParam)
}

func VersionRoute(s string) string {
	return fmt.Sprintf(s, VersionParam)
}
