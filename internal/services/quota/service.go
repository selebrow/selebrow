package quota

import (
	"github.com/selebrow/selebrow/pkg/dto"
	"github.com/selebrow/selebrow/pkg/quota"
)

type QuotaService interface {
	GetQuotaUsage() *dto.QuotaUsage
}

type QuotaServiceImpl struct {
	qa quota.QuotaAuthorizer
}

func NewQuotaService(qa quota.QuotaAuthorizer) *QuotaServiceImpl {
	return &QuotaServiceImpl{qa: qa}
}

func (q *QuotaServiceImpl) GetQuotaUsage() *dto.QuotaUsage {
	if !q.qa.Enabled() {
		return nil
	}

	return &dto.QuotaUsage{
		Limit:     q.qa.Limit(),
		Allocated: q.qa.Allocated(),
	}
}
