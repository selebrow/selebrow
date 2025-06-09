package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/selebrow/selebrow/pkg/models"
)

func ErrorHandler(err error, c echo.Context) {
	httpErr := &echo.HTTPError{}
	if errors.As(err, &httpErr) {
		c.Echo().DefaultHTTPErrorHandler(err, c)
		return
	}

	if c.Response().Committed {
		return
	}

	w3cErr := &models.W3CError{}
	if errors.As(err, &w3cErr) {
		_ = c.JSON(w3cErr.Code(), w3cErr)
		return
	}

	code := http.StatusInternalServerError
	if e, ok := unwrapErrorWithCode(err); ok {
		code = e.Code()
	}
	_ = c.String(code, err.Error())
}

func unwrapErrorWithCode(err error) (models.ErrorWithCode, bool) {
	var e models.ErrorWithCode
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
