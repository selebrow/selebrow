package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
)

var configData = map[string]string{
	"abc": "123",
	"qqq": "test",
}

func TestConfigController_List(t *testing.T) {
	g := NewWithT(t)

	cc := NewConfigController(configData)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/config", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := cc.List(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	g.Expect(rec.Body.String()).To(MatchJSON(`{
  "files": {
    "abc": {
      "sha256Sum": "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
    },
    "qqq": {
      "sha256Sum": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
    }
  }
}`))
}

func TestConfigController_GetConfig(t *testing.T) {
	g := NewWithT(t)

	cc := NewConfigController(configData)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/config/qqq", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues("qqq")

	err := cc.GetConfig(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	g.Expect(rec.Body.String()).To(Equal("test"))
}

func TestConfigController_GetConfig_NotFound(t *testing.T) {
	g := NewWithT(t)

	cc := NewConfigController(configData)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/config/zzz", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues("zzz")

	err := cc.GetConfig(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusNotFound))
}
