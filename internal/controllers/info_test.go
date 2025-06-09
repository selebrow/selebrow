package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/pkg/dto"
)

func TestInfoController_Info(t *testing.T) {
	g := NewWithT(t)

	ic := NewInfoController("app", "dev", "deadbeef")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/info", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ic.Info(c)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rec).To(HaveHTTPStatus(http.StatusOK))

	var gotResp dto.AppInfo
	err = json.NewDecoder(rec.Body).Decode(&gotResp)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(gotResp).To(Equal(dto.AppInfo{
		Name:   "app",
		GitRef: "dev",
		GitSha: "deadbeef",
	}))
}
