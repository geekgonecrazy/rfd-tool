package renderer

import (
	"bytes"
	"io"

	"github.com/adrg/frontmatter"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/renderer/d2"
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
			&d2.Extender{}, // Add d2 before other extensions
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

	// Use a temporary struct for YAML parsing
	type yamlMeta struct {
		Title      string          `yaml:"title"`
		Authors    []string        `yaml:"authors"`
		State      models.RFDState `yaml:"state"`
		Discussion string          `yaml:"discussion"`
		Tags       []string        `yaml:"tags"`
		Public     bool            `yaml:"public"`
	}

	var meta yamlMeta

	body, err := frontmatter.Parse(f, &meta)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return nil, err
	}

	// Convert to RFDMeta
	rfd.ID = rfdNum
	rfd.RFDMeta = models.RFDMeta{
		Title:      meta.Title,
		Authors:    []models.Author{}, // Will be populated by core after processing
		State:      meta.State,
		Discussion: meta.Discussion,
		Tags:       meta.Tags,
		Public:     meta.Public,
	}
	rfd.ContentMD = string(body)
	rfd.Content = string(buf.Bytes())

	// Store the author strings temporarily for core to process
	rfd.AuthorStrings = meta.Authors

	return &rfd, nil
}
