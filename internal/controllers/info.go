package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/selebrow/selebrow/pkg/dto"
)

type InfoController struct {
	AppName string
	GitRef  string
	GitSha  string
}

func NewInfoController(appName string, gitRef string, gitSha string) *InfoController {
	return &InfoController{AppName: appName, GitRef: gitRef, GitSha: gitSha}
}

func (i *InfoController) Info(c echo.Context) error {
	info := &dto.AppInfo{
		Name:   i.AppName,
		GitRef: i.GitRef,
		GitSha: i.GitSha,
	}
	return c.JSON(http.StatusOK, info)
}
