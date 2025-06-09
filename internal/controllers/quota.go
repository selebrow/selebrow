package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/selebrow/selebrow/internal/services/quota"
	"github.com/selebrow/selebrow/pkg/models"
)

var errQuotaUnavailable = errors.New("quota information is not available")

type QuotaController struct {
	srv quota.QuotaService
}

func NewQuotaController(srv quota.QuotaService) *QuotaController {
	return &QuotaController{srv: srv}
}

func (q *QuotaController) QuotaUsage(c echo.Context) error {
	usage := q.srv.GetQuotaUsage()
	if usage == nil {
		return models.NewErrorMessage(http.StatusServiceUnavailable, errQuotaUnavailable)
	}
	return c.JSON(http.StatusOK, usage)
}
