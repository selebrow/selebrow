package log

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetLogger(t *testing.T) {
	g := NewWithT(t)
	l1 := GetLogger()
	g.Expect(l1).ToNot(BeNil())

	l2 := GetLogger()
	g.Expect(l2).To(BeIdenticalTo(l1))
}

func TestNewConsoleLogger_JSON(t *testing.T) {
	g := NewWithT(t)
	out := path.Join(t.TempDir(), "out.log")
	t.Setenv("SB_LOG_OUTPUT", out)
	t.Setenv("SB_LOG_FORMAT", "jSoN")

	l := NewConsoleLogger()
	l.Debug("debug ignored")
	l.Info("test")
	_ = l.Sync()

	got, err := os.ReadFile(out)
	g.Expect(err).ToNot(HaveOccurred())

	var val map[string]interface{}
	err = json.NewDecoder(bytes.NewBuffer(got)).Decode(&val)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(err)
	g.Expect(val["message"]).To(Equal("test"))
}

func TestNewConsoleLogger_Text(t *testing.T) {
	g := NewWithT(t)
	out := path.Join(t.TempDir(), "out.log")
	t.Setenv("SB_LOG_OUTPUT", out)
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	l := NewConsoleLogger()
	l.Debug("debug ignored")
	l.Info("test")
	_ = l.Sync()

	got, err := os.ReadFile(out)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(string(got)).To(HaveSuffix("test\n"))
}

func TestNewConsoleLogger_Debug(t *testing.T) {
	g := NewWithT(t)
	out := path.Join(t.TempDir(), "out.log")
	t.Setenv("SB_LOG_OUTPUT", out)
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("SB_LOG_LEVEL", "Debug")

	l := NewConsoleLogger()
	l.Debug("test debug")
	_ = l.Sync()

	got, err := os.ReadFile(out)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(string(got)).To(HaveSuffix("test debug\n"))
}
