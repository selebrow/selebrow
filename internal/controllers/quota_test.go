package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/models"
)

func TestQuotaController_QuotaUsage(t *testing.T) {
	g := NewWithT(t)

	qs := new(mocks.QuotaService)
	qc := NewQuotaController(qs)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/quota", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expResp := &dto.QuotaUsage{
		Limit:     123,
		Allocated: 44,
	}
	qs.EXPECT().GetQuotaUsage().Return(expResp)
	err := qc.QuotaUsage(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	var gotResp dto.QuotaUsage
	err = json.NewDecoder(rec.Body).Decode(&gotResp)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(gotResp).To(Equal(*expResp))

	qs.AssertExpectations(t)
}

func TestQuotaController_QuotaUsageNotAvailable(t *testing.T) {
	g := NewWithT(t)

	qs := new(mocks.QuotaService)
	qc := NewQuotaController(qs)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/quota", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	qs.EXPECT().GetQuotaUsage().Return(nil)
	err := qc.QuotaUsage(c)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusServiceUnavailable))

	qs.AssertExpectations(t)
}
