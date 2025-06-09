package controllers

import (
	"net/http"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/pkg/browsers"
	"github.com/selebrow/selebrow/pkg/models"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type BrowsersCatalogController struct {
	cat browsers.BrowsersCatalog
}

func NewBrowsersCatalogController(cat browsers.BrowsersCatalog) *BrowsersCatalogController {
	return &BrowsersCatalogController{cat: cat}
}

func (b *BrowsersCatalogController) Browsers(c echo.Context) error {
	var p = models.BrowserProtocol(c.QueryParam(router.ProtoQParam))
	if p == "" {
		p = models.WebdriverProtocol
	}
	br := b.cat.GetBrowsers(p, c.QueryParam(router.FlavorQParam))
	if br == nil {
		return models.NewErrorMessage(http.StatusNotFound, errors.Errorf("no browsers configured for protocol %v", p))
	}
	return c.JSON(http.StatusOK, br)
}
