package html

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"sync"

	"github.com/Masterminds/sprig/v3"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

//go:embed templates
var templatesEmbedFS embed.FS

func TemplatesFS() fs.FS {
	if devMode {
		return devFS
	}
	return templatesEmbedFS
}

type TemplateRenderer struct {
	templates *template.Template
	fSys      fs.FS
	m         sync.Mutex
}

func NewTemplateRenderer(fSys fs.FS) (*TemplateRenderer, error) {
	templates, err := initTemplates(fSys)
	if err != nil {
		return nil, err
	}
	return &TemplateRenderer{
		templates: templates,
		fSys:      fSys,
	}, nil
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	if devMode {
		t.m.Lock()
		defer t.m.Unlock()
		templates, err := initTemplates(t.fSys)
		if err != nil {
			return errors.Wrap(err, "failed to reload templates in dev mode")
		}
		t.templates = templates
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func initTemplates(fSys fs.FS) (*template.Template, error) {
	tmpl, err := template.New("html").
		Funcs(sprig.HtmlFuncMap()).
		ParseFS(fSys, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
