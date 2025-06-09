package capabilities_test

import (
	"hash"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/capabilities"
)

const (
	fullW3c = `{
    "capabilities":
    {
        "alwaysMatch":
        {
            "browserName": "firefox",
            "browserVersion": "119.0",
            "platformName": "gnu/hurd",
            "selenoid:options":
            {
                "sessionTimeout": "10m",
                "enableVNC": true,                
                "env": [ "a=b" ],
                "flavor": "test"
            }
        },
        "firstMatch":
        [
            {
                "selenoid:options":
                {
                    "sessionTimeout": "10m",
                    "name": "my-test",
                    "flavor": "ignore"
                }
            },
            {
                "selenoid:options":
                {
                    "screenResolution": "1920x1080x24",
                    "flavor": "ignore2"
                }
            }
        ]
    },
    "desiredCapabilities": {}
}`
	fullJsonWire = `{
    "desiredCapabilities":
    {
        "browserName": "firefox",
        "version": "119.0",
		"platform": "cp/m",
        "selenoid:options":
        {
            "screenResolution": "320x200x8",
            "sessionTimeout": "11m",
			"name": "my-test-jsw",
			"flavor": "testjsw",
			"env": [ "c=d" ],
            "enableVNC": true
        }
    }
}`
)

func TestNewCapabilities(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expName       string
		expVersion    string
		expPlatform   string
		expResolution string
		expFlavor     string
		expTimeout    time.Duration
		expVnc        bool
		expTestName   string
		expEnvs       []string
		wantErr       bool
	}{
		{
			name:    "Malformed capabilities",
			input:   `{ qqq`,
			wantErr: true,
		},
		{
			name:    "No supported capabilities",
			input:   `{ "qqq": 123 }`,
			wantErr: true,
		},
		{
			name:          "W3C full",
			input:         fullW3c,
			expName:       "firefox",
			expPlatform:   "gnu/hurd",
			expVersion:    "119.0",
			expResolution: "1920x1080x24",
			expFlavor:     "test",
			expTimeout:    10 * time.Minute,
			expVnc:        true,
			expTestName:   "my-test",
			expEnvs:       []string{"a=b"},
		},
		{
			name:    "W3C minimal",
			input:   `{"capabilities":{"alwaysMatch":{}}}`,
			expEnvs: []string{},
		},
		{
			name:    "W3C malformed resolution",
			input:   `{"capabilities":{"alwaysMatch":{"selenoid:options":{"screenResolution":"1920x1080"}}}}`,
			wantErr: true,
		},
		{
			name:          "JsonWire full",
			input:         fullJsonWire,
			expName:       "firefox",
			expPlatform:   "cp/m",
			expVersion:    "119.0",
			expResolution: "320x200x8",
			expFlavor:     "testjsw",
			expTimeout:    11 * time.Minute,
			expVnc:        true,
			expTestName:   "my-test-jsw",
			expEnvs:       []string{"c=d"},
		},
		{
			name:    "JsonWire minimal",
			input:   `{"desiredCapabilities":{}}`,
			expEnvs: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			got, err := capabilities.NewCapabilities(strings.NewReader(tt.input))
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(got.GetRawCapabilities()).To(Equal([]byte(tt.input)))
				g.Expect(got.GetName()).To(Equal(tt.expName))
				g.Expect(got.GetPlatform()).To(Equal(tt.expPlatform))
				g.Expect(got.GetVersion()).To(Equal(tt.expVersion))
				g.Expect(got.GetResolution()).To(Equal(tt.expResolution))
				g.Expect(got.GetFlavor()).To(Equal(tt.expFlavor))
				g.Expect(got.GetTimeout()).To(Equal(tt.expTimeout))
				g.Expect(got.IsVNCEnabled()).To(Equal(tt.expVnc))
				g.Expect(got.GetTestName()).To(Equal(tt.expTestName))
				g.Expect(got.GetEnvs()).To(Equal(tt.expEnvs))
			}
		})
	}
}

func TestGetHash(t *testing.T) {
	g := NewWithT(t)

	caps := new(mocks.Capabilities)
	h := new(mocks.Hash)

	capabilities.NewHash = func() hash.Hash {
		return h
	}

	caps.EXPECT().GetPlatform().Return("cp/m").Once()
	caps.EXPECT().GetName().Return("mosaic").Once()
	caps.EXPECT().GetFlavor().Return("experimental").Once()
	caps.EXPECT().GetVersion().Return("0.1").Once()
	caps.EXPECT().GetResolution().Return("320x200x8").Once()
	caps.EXPECT().GetEnvs().Return([]string{"k1=v1", "k2=v2"}).Once()
	caps.EXPECT().IsVNCEnabled().Return(true).Once()
	caps.EXPECT().GetLabels().Return(map[string]string{"l2": "v1", "l1": "v2"}).Once()
	caps.EXPECT().GetLinks().Return([]string{"link1", "link2"}).Once()
	caps.EXPECT().GetHosts().Return([]string{"host1", "host2"}).Once()
	caps.EXPECT().GetNetworks().Return([]string{"n2", "n1"}).Once()

	h.EXPECT().Write([]byte("cp/m")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("mosaic")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("experimental")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("0.1")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("320x200x8")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("k1=v1;k2=v2")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("true")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("l1=v2;l2=v1")).Return(0, nil)
	h.EXPECT().Write([]byte("link1;link2")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("host1;host2")).Return(0, nil).Once()
	h.EXPECT().Write([]byte("n1;n2")).Return(0, nil).Once()
	h.EXPECT().Sum([]byte(nil)).Return([]byte("test")).Once()

	got := capabilities.GetHash(caps)
	g.Expect(got).To(Equal([]byte("test")))

	caps.AssertExpectations(t)
	h.AssertExpectations(t)
}
