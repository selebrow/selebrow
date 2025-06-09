package html

import (
	"embed"
	"testing"

	. "github.com/onsi/gomega"
)

func TestStaticFS(t *testing.T) {
	g := NewWithT(t)

	devMode = false
	f := StaticFS()
	_, ok := f.(embed.FS)
	g.Expect(ok).To(BeTrue())
}

func TestStaticFS_DevMode(t *testing.T) {
	g := NewWithT(t)

	devMode = true
	f := StaticFS()
	_, ok := f.(embed.FS)
	g.Expect(ok).To(BeFalse())
}
