package renderer

import (
	"bytes"
	"io"

	"github.com/adrg/frontmatter"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
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
			&anchor.Extender{
				Texter:   anchor.Text("#"),
				Position: anchor.After,
			},
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(false),
				),
			),
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

	rfdMeta := models.RFDMeta{}

	body, err := frontmatter.Parse(f, &rfdMeta)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return nil, err
	}

	rfd.ID = rfdNum
	rfd.RFDMeta = rfdMeta
	rfd.ContentMD = string(body)
	rfd.Content = string(buf.Bytes())

	return &rfd, nil
}
