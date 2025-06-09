package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/selebrow/selebrow/pkg/models"
)

type WDStatusController struct{}

func NewWDStatusController() *WDStatusController {
	return &WDStatusController{}
}

func (*WDStatusController) Status(c echo.Context) error {
	return c.JSON(http.StatusOK, models.NewWebDriverStatus(true))
}
