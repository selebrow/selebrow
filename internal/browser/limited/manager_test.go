package limited

import (
	"context"
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

const (
	testProt models.BrowserProtocol = "test"
	testPort models.ContainerPort   = "port"
)

var (
	testUrl, _   = url.Parse("http:/test")
	testHost     = "test:123"
	testHostPort = "qqqq:789"
)

func TestLimitedBrowserManager_Allocate(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(mgr *mocks.BrowserManager, qa *mocks.QuotaAuthorizer, caps *mocks.Capabilities)
		wantErr    bool
	}{
		{
			name: "Happy path",
			setupMocks: func(mgr *mocks.BrowserManager, qa *mocks.QuotaAuthorizer, caps *mocks.Capabilities) {
				qa.EXPECT().Reserve(mock.Anything).Return(nil).Once()
				br := new(mocks.Browser)
				mgr.EXPECT().Allocate(context.TODO(), testProt, caps).Return(br, nil).Once()
				br.EXPECT().GetURL().Return(testUrl).Once()
				br.EXPECT().GetHost().Return(testHost).Once()
				br.EXPECT().GetHostPort(testPort).Return(testHostPort).Once()
				br.EXPECT().Close(context.TODO(), true).Once()
				qa.EXPECT().Release().Return(0).Once()
			},
		},
		{
			name: "No quota",
			setupMocks: func(_ *mocks.BrowserManager, qa *mocks.QuotaAuthorizer, _ *mocks.Capabilities) {
				qa.EXPECT().Reserve(mock.Anything).Return(errors.New("test error")).Once()
			},
			wantErr: true,
		},
		{
			name: "Allocate error",
			setupMocks: func(mgr *mocks.BrowserManager, qa *mocks.QuotaAuthorizer, caps *mocks.Capabilities) {
				qa.EXPECT().Reserve(mock.Anything).Return(nil).Once()
				mgr.EXPECT().Allocate(context.TODO(), testProt, caps).Return(nil, errors.New("test error")).Once()
				qa.EXPECT().Release().Return(0).Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			mgr := new(mocks.BrowserManager)
			qa := new(mocks.QuotaAuthorizer)
			caps := new(mocks.Capabilities)

			tt.setupMocks(mgr, qa, caps)
			m := NewLimitedBrowserManager(mgr, qa, time.Minute, zaptest.NewLogger(t))
			got, err := m.Allocate(context.TODO(), testProt, caps)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(got).ToNot(BeNil())
				g.Expect(got.GetURL()).To(Equal(testUrl))
				g.Expect(got.GetHost()).To(Equal(testHost))
				g.Expect(got.GetHostPort(testPort)).To(Equal(testHostPort))
				got.Close(context.TODO(), true)
			}
			mgr.AssertExpectations(t)
			qa.AssertExpectations(t)
		})
	}
}
