package config

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestParseCmdLine(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("HOME", "")
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("CI", "yes")

	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f, exit, err := ParseCmdLine(f, []string{"--namespace=qqq", "--backend=kubernetes"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exit).To(BeFalse())

	g.Expect(f.GetString("namespace")).To(Equal("qqq"))
	g.Expect(f.GetString("kube-config")).To(BeEmpty())
	g.Expect(f.GetBool("kube-cluster-mode-out")).To(BeFalse())
	g.Expect(f.GetBool("ui")).To(BeFalse())
}

func TestParseCmdLine_Error(t *testing.T) {
	g := NewWithT(t)
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	_, exit, err := ParseCmdLine(f, []string{"--nonexistent"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(exit).To(BeTrue())
}

func TestParseCmdLine_DefaultKubeConfig(t *testing.T) {
	g := NewWithT(t)
	err := os.Setenv("HOME", "/userhome")
	g.Expect(err).ToNot(HaveOccurred())
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f, exit, err := ParseCmdLine(f, []string{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exit).To(BeFalse())

	g.Expect(f.GetString("kube-config")).To(Equal("/userhome/.kube/config"))
}

func TestParseCmdLine_InClusterMode(t *testing.T) {
	g := NewWithT(t)
	err := os.Setenv("KUBERNETES_SERVICE_HOST", "qwe")
	g.Expect(err).ToNot(HaveOccurred())
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f, exit, err := ParseCmdLine(f, []string{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exit).To(BeFalse())

	g.Expect(f.GetBool("kube-cluster-mode-out")).To(BeFalse())
}

func TestParseCmdLine_Help(t *testing.T) {
	g := NewWithT(t)
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	_, exit, err := ParseCmdLine(f, []string{"-h"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exit).To(BeTrue())
}
