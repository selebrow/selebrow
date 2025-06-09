package capabilities

import (
	"encoding/json"
	"hash/fnv"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/selebrow/selebrow/pkg/models"
)

var NewHash = fnv.New128a

type Capabilities interface {
	GetName() string
	GetVersion() string
	GetPlatform() string
	GetResolution() string
	IsVNCEnabled() bool
	GetTestName() string
	GetEnvs() []string
	GetTimeout() time.Duration
	GetRawCapabilities() []byte
	GetFlavor() string
	GetLinks() []string
	GetHosts() []string
	GetNetworks() []string
	GetLabels() map[string]string
}

type CapsWrapper struct {
	DesiredCapabilities *models.JsonWireCapabilities `json:"desiredCapabilities,omitempty"`
	Capabilities        *models.W3CCapabilities      `json:"capabilities,omitempty"`
}

func NewCapabilities(r io.Reader) (Capabilities, error) {
	var caps CapsWrapper

	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read Capabilities data")
	}

	err = json.Unmarshal(raw, &caps)
	if err != nil {
		return nil, errors.Wrap(err, "Failed parsing Capabilities json")
	}

	var c Capabilities
	if caps.Capabilities == nil {
		if caps.DesiredCapabilities == nil {
			return nil, errors.New("No valid capabilities provided in request")
		}

		caps.DesiredCapabilities.RawCapabilities = raw
		c = caps.DesiredCapabilities // JSONWire caps
	} else {
		w3cCaps := caps.Capabilities
		w3cCaps.Merge()
		w3cCaps.RawCapabilities = raw
		c = w3cCaps // W3C caps
	}

	if err := validateCaps(c); err != nil {
		return nil, err
	}
	return c, nil
}

func GetHash(caps Capabilities) []byte {
	hash := NewHash()
	hash.Write([]byte(caps.GetPlatform()))
	hash.Write([]byte(caps.GetName()))
	hash.Write([]byte(caps.GetFlavor()))
	hash.Write([]byte(caps.GetVersion()))
	hash.Write([]byte(caps.GetResolution()))
	hash.Write([]byte(joinStrings(caps.GetEnvs())))
	hash.Write([]byte(strconv.FormatBool(caps.IsVNCEnabled())))
	hash.Write([]byte(joinStringMap(caps.GetLabels())))
	hash.Write([]byte(joinStrings(caps.GetLinks())))
	hash.Write([]byte(joinStrings(caps.GetHosts())))
	hash.Write([]byte(joinStrings(caps.GetNetworks())))
	return hash.Sum(nil)
}

func joinStrings(s []string) string {
	res := slices.Clone(s)
	slices.Sort(res)
	return strings.Join(res, ";")
}

func joinStringMap(s map[string]string) string {
	if len(s) == 0 {
		return ""
	}
	keys := maps.Keys(s)
	var sb strings.Builder
	for i, k := range slices.Sorted(keys) {
		if i > 0 {
			sb.WriteRune(';')
		}
		sb.WriteString(k)
		sb.WriteRune('=')
		sb.WriteString(s[k])
	}
	return sb.String()
}

//nolint:gocritic // gocritic suggests wrong regex
var resolutionRegex = regexp.MustCompile(`^(|\d+x\d+x\d+)$`)

func validateCaps(caps Capabilities) error {
	if res := caps.GetResolution(); !resolutionRegex.MatchString(res) {
		return errors.Errorf("incorrect resolution value format (expected WIDTHxHEIGHTxBPP)")
	}
	return nil
}
