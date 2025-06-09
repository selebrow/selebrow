package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/pkg/models"
)

func TestErrorHandler_HTTPError(t *testing.T) {
	g := NewWithT(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/whatever", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := &echo.HTTPError{
		Code:     http.StatusNotImplemented,
		Message:  "test error",
		Internal: nil,
	}
	ErrorHandler(err, c)

	g.Expect(rec).To(HaveHTTPStatus(http.StatusNotImplemented))
	g.Expect(rec.Body.String()).To(MatchJSON(`{"message": "test error"}`))
}

func TestErrorHandler_W3CErr(t *testing.T) {
	g := NewWithT(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/whatever", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := models.NewW3CErr(http.StatusConflict, "error", errors.New("test error"))
	ErrorHandler(err, c)

	g.Expect(rec).To(HaveHTTPStatus(http.StatusConflict))
	g.Expect(rec.Body.String()).To(MatchJSON(`{
              "value": {
                "error": "error",
                "message": "test error",
                "stacktrace": "test error"
              }
            }`))
}

func TestErrorHandler_ErrorWithCode(t *testing.T) {
	g := NewWithT(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/whatever", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := models.NewErrorMessage(http.StatusConflict, errors.New("test error"))
	ErrorHandler(err, c)

	g.Expect(rec).To(HaveHTTPStatus(http.StatusConflict))
	g.Expect(rec.Body.String()).To(Equal("test error"))
}

func TestErrorHandler_Default(t *testing.T) {
	g := NewWithT(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/whatever", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := errors.New("test error")
	ErrorHandler(err, c)

	g.Expect(rec).To(HaveHTTPStatus(http.StatusInternalServerError))
	g.Expect(rec.Body.String()).To(Equal("test error"))
}

func TestErrorHandler_Committed(t *testing.T) {
	g := NewWithT(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/whatever", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	rec.WriteHeader(http.StatusNotFound)
	c.Response().Committed = true
	err := errors.New("test error")
	ErrorHandler(err, c)

	g.Expect(rec).To(HaveHTTPStatus(http.StatusNotFound))
	g.Expect(rec.Body.String()).To(BeEmpty())
}
