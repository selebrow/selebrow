package quota

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/dto"
)

func TestQuotaService_NoQuota(t *testing.T) {
	g := NewWithT(t)

	qa := new(mocks.QuotaAuthorizer)
	qa.EXPECT().Enabled().Return(false).Once()
	s := NewQuotaService(qa)

	got := s.GetQuotaUsage()
	g.Expect(got).To(BeNil())
	qa.AssertExpectations(t)
}

func TestQuotaService_GetQuotaUsage(t *testing.T) {
	g := NewWithT(t)

	qa := new(mocks.QuotaAuthorizer)
	qa.EXPECT().Enabled().Return(true).Once()
	qa.EXPECT().Limit().Return(11).Once()
	qa.EXPECT().Allocated().Return(5).Once()

	s := NewQuotaService(qa)

	got := s.GetQuotaUsage()
	g.Expect(got).To(Equal(&dto.QuotaUsage{
		Limit:     11,
		Allocated: 5,
	}))

	qa.AssertExpectations(t)
}
