package renderer

import (
	"bytes"
	"io"

	"github.com/adrg/frontmatter"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/mermaid"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&mermaid.Extender{
				//RenderMode: mermaid.RenderModeServer,
			},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
}

func RenderRFD(rfdNum string, f io.Reader) (*models.RFD, error) {
	rfd := models.RFD{}

	body, err := frontmatter.Parse(f, &rfd)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return nil, err
	}

	rfd.ID = rfdNum
	rfd.ContentMD = string(body)
	rfd.Content = string(buf.Bytes())

	return &rfd, nil
}
