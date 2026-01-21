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

	"github.com/go-viper/mapstructure/v2"
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
	*models.JsonWireCapabilities
	Capabilities *models.W3CCapabilities `json:"capabilities,omitempty"`
}

func NewCapabilities(r io.Reader, defaultProxy *models.ProxyOptions) (Capabilities, error) {
	var unparsedCaps CapsWrapper

	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Capabilities data")
	}

	// Deserialize to map[string]interface{} structures
	if err := json.Unmarshal(raw, &unparsedCaps); err != nil {
		return nil, errors.Wrap(err, "failed parsing Capabilities json")
	}

	c := new(models.Capabilities)

	var updateProxyFn func(proxy *models.ProxyOptions)
	if unparsedCaps.Capabilities == nil {
		if unparsedCaps.JsonWireCapabilities == nil || unparsedCaps.DesiredCapabilities == nil {
			return nil, errors.New("no valid capabilities provided in request")
		}

		if err := decodeCapabilities(c, unparsedCaps.DesiredCapabilities, "jsonwire"); err != nil {
			return nil, errors.Wrap(err, "failed to decode JsonWire capabilities")
		}
		//nolint:staticcheck // QF1008: we want to be more explicit here
		updateProxyFn = unparsedCaps.JsonWireCapabilities.UpdateProxy
	} else {
		if err := decodeCapabilities(c, unparsedCaps.Capabilities.Merge(), "w3c"); err != nil {
			return nil, errors.Wrap(err, "failed to decode W3C capabilities")
		}
		updateProxyFn = unparsedCaps.Capabilities.UpdateProxy
	}

	if defaultProxy != nil && (c.Proxy == nil || c.Proxy.ProxyType != models.ProxyTypeManual) {
		updateProxyFn(defaultProxy)
		c.Proxy = defaultProxy
		raw, err = json.Marshal(unparsedCaps)
		if err != nil {
			return nil, errors.Wrap(err, "failed to serialize updated capabilities")
		}
	}

	c.RawCapabilities = raw

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

func decodeCapabilities(caps Capabilities, data map[string]interface{}, tag string) error {
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.TextUnmarshallerHookFunc(),
		Result:     caps,
		TagName:    tag,
	})
	if err != nil {
		return err
	}

	return d.Decode(data)
}
