package d2

import (
	"bytes"
	"context"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// Kind is the kind of a D2 block.
var Kind = ast.NewNodeKind("D2Block")

// Block represents a D2 code block.
type Block struct {
	ast.BaseBlock
}

// Kind returns the kind of this node.
func (n *Block) Kind() ast.NodeKind {
	return Kind
}

// Dump dumps the block to stdout.
func (n *Block) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// Transformer transforms d2 code blocks to Block nodes.
type Transformer struct{}

// Transform transforms the given Markdown AST.
func (t *Transformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		cb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		lang := cb.Language(reader.Source())
		if !bytes.Equal(lang, []byte("d2")) {
			return ast.WalkContinue, nil
		}

		// Replace the fenced code block with a D2 block
		d2Block := &Block{}
		d2Block.SetLines(cb.Lines())

		parent := node.Parent()
		parent.ReplaceChild(parent, node, d2Block)

		return ast.WalkContinue, nil
	})
}

// Renderer renders D2 blocks as SVG.
type Renderer struct{}

// RegisterFuncs registers the renderer funcs for D2.
func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(Kind, r.Render)
}

// Render renders a D2 block.
func (r *Renderer) Render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	block := node.(*Block)
	
	// Extract D2 source code
	var d2Source bytes.Buffer
	lines := block.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		d2Source.Write(line.Value(source))
	}

	// Render D2 to SVG
	svg, err := renderD2ToSVG(d2Source.String())
	if err != nil {
		// Fallback: render as a code block with error
		w.WriteString(`<pre class="d2-error"><code>`)
		w.WriteString(fmt.Sprintf("D2 rendering error: %v\n\n", err))
		w.Write(util.EscapeHTML(d2Source.Bytes()))
		w.WriteString(`</code></pre>`)
		return ast.WalkContinue, nil
	}

	// Write SVG wrapped in a div for styling
	w.WriteString(`<div class="d2-diagram">`)
	w.Write(svg)
	w.WriteString(`</div>`)

	return ast.WalkContinue, nil
}

// renderD2ToSVG converts D2 source to SVG
func renderD2ToSVG(d2Source string) ([]byte, error) {
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, fmt.Errorf("failed to create ruler: %w", err)
	}

	ctx := context.Background()
	layout := "dagre"
	
	diagram, _, err := d2lib.Compile(ctx, d2Source, &d2lib.CompileOptions{
		Layout: &layout,
		Ruler: ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}, &d2svg.RenderOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to compile D2: %w", err)
	}

	svg, err := d2svg.Render(diagram, &d2svg.RenderOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to render SVG: %w", err)
	}

	return svg, nil
}

// Extender extends goldmark with D2 support.
type Extender struct{}

// Extend extends the goldmark processor with D2 support.
func (e *Extender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&Transformer{}, 200), // Higher priority than syntax highlighting
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&Renderer{}, 200), // Higher priority than syntax highlighting
		),
	)
}