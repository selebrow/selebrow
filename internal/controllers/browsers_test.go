package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/internal/router"
	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/models"
)

var (
	testBrowsers = []dto.Browser{
		{
			Name:            "br1",
			DefaultVersion:  "1",
			DefaultPlatform: "rt11/sj",
			Versions: []dto.BrowserVersion{
				{
					Number:   "1",
					Platform: "rt11/sj",
				},
			},
		},
	}

	expBrowsersResp = `[
              {
                "Name": "br1",
                "DefaultVersion": "1",
                "DefaultPlatform": "rt11/sj",
                "Versions": [
                  {
                    "Number": "1",
                    "Platform": "rt11/sj"
                  }
                ]
              }
            ]`
)

func TestBrowsersCatalogController_Browsers(t *testing.T) {
	g := NewWithT(t)

	cat := new(mocks.BrowsersCatalog)
	bc := NewBrowsersCatalogController(cat)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/browsers", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.QueryParams().Set(router.ProtoQParam, "test")
	c.QueryParams().Set(router.FlavorQParam, "some")

	cat.EXPECT().GetBrowsers(models.BrowserProtocol("test"), "some").Return(testBrowsers)
	err := bc.Browsers(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))
	g.Expect(rec.Body.String()).To(MatchJSON(expBrowsersResp))

	cat.AssertExpectations(t)
}

func TestBrowsersCatalogController_Browsers_Default_Proto(t *testing.T) {
	g := NewWithT(t)

	cat := new(mocks.BrowsersCatalog)
	bc := NewBrowsersCatalogController(cat)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/browsers", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cat.EXPECT().GetBrowsers(models.WebdriverProtocol, "").Return(testBrowsers)
	err := bc.Browsers(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))
	g.Expect(rec.Body.String()).To(MatchJSON(expBrowsersResp))

	cat.AssertExpectations(t)
}

func TestBrowsersCatalogController_Browsers_Not_Found(t *testing.T) {
	g := NewWithT(t)

	cat := new(mocks.BrowsersCatalog)
	bc := NewBrowsersCatalogController(cat)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/browsers", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cat.EXPECT().GetBrowsers(models.WebdriverProtocol, "").Return(nil)
	err := bc.Browsers(c)
	g.Expect(err).To(MatchError("no browsers configured for protocol webdriver"))
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusNotFound))

	cat.AssertExpectations(t)
}
