package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/pkg/models"
)

func TestWDStatusController_Status(t *testing.T) {
	g := NewWithT(t)

	sc := NewWDStatusController()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/stat", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := sc.Status(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	var gotResp models.WebDriverStatus
	err = json.NewDecoder(rec.Body).Decode(&gotResp)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(gotResp).To(Equal(models.WebDriverStatus{
		Value: models.WebDriverReadyStatus{
			Ready: true,
		},
	}))
}
