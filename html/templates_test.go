package html

import (
	"embed"
	"strings"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"
)

type testData struct {
	Test string
}

func TestTemplateRenderer_Render(t *testing.T) {
	g := NewWithT(t)
	devMode = false
	testFs := fstest.MapFS{
		"templates/test.tmpl": {
			Data: []byte("{{ title .Test }}"),
		},
	}

	r, err := NewTemplateRenderer(testFs)
	g.Expect(err).ToNot(HaveOccurred())

	sb := strings.Builder{}
	err = r.Render(&sb, "test.tmpl", &testData{Test: "test1"}, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sb.String()).To(Equal("Test1"))

	testFs["templates/test.tmpl"].Data = []byte("changed")

	sb = strings.Builder{}
	err = r.Render(&sb, "test.tmpl", &testData{Test: "test1"}, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sb.String()).To(Equal("Test1"))
}

func TestTemplateRenderer_Render_DevMode(t *testing.T) {
	g := NewWithT(t)
	devMode = true
	testFs := fstest.MapFS{
		"templates/test.tmpl": {
			Data: []byte("{{ title .Test }}"),
		},
	}

	r, err := NewTemplateRenderer(testFs)
	g.Expect(err).ToNot(HaveOccurred())

	sb := strings.Builder{}
	err = r.Render(&sb, "test.tmpl", &testData{Test: "test1"}, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sb.String()).To(Equal("Test1"))

	testFs["templates/test.tmpl"].Data = []byte("changed {{ bad")

	sb = strings.Builder{}
	err = r.Render(&sb, "test.tmpl", &testData{}, nil)
	g.Expect(err).To(HaveOccurred())

	testFs["templates/test.tmpl"].Data = []byte("changed {{ .Test }}")

	sb = strings.Builder{}
	err = r.Render(&sb, "test.tmpl", &testData{Test: "test1"}, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sb.String()).To(Equal("changed test1"))
}

func TestNewTemplateRenderer_Negative(t *testing.T) {
	g := NewWithT(t)
	testFs := fstest.MapFS{}

	_, err := NewTemplateRenderer(testFs)
	g.Expect(err).To(HaveOccurred())
}

func TestTemplatesFS(t *testing.T) {
	g := NewWithT(t)

	devMode = false
	f := TemplatesFS()
	_, ok := f.(embed.FS)
	g.Expect(ok).To(BeTrue())
}

func TestTemplatesFS_DevMode(t *testing.T) {
	g := NewWithT(t)

	devMode = true
	f := TemplatesFS()
	_, ok := f.(embed.FS)
	g.Expect(ok).To(BeFalse())
}
