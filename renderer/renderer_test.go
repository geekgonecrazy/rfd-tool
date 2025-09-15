package renderer

import (
	"strings"
	"testing"
)

func TestRenderRFDWithDiagrams(t *testing.T) {
	// Test content with both mermaid and d2 diagrams
	content := `---
title: Test RFD
authors: ["test@example.com"]
state: discussion
---

# Test RFD with Diagrams

This RFD contains both mermaid and d2 diagrams.

## Mermaid Diagram

` + "```mermaid" + `
graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Action 1]
    B -->|No| D[Action 2]
` + "```" + `

## D2 Diagram

` + "```d2" + `
x -> y: hello world
y -> z: here's a second line
z -> x
` + "```" + `

## Conclusion

Both diagrams should render properly.
`

	reader := strings.NewReader(content)
	rfd, err := RenderRFD("001", reader)
	if err != nil {
		t.Fatalf("Failed to render RFD: %v", err)
	}

	if rfd.ID != "001" {
		t.Errorf("Expected ID '001', got '%s'", rfd.ID)
	}

	if rfd.Title != "Test RFD" {
		t.Errorf("Expected title 'Test RFD', got '%s'", rfd.Title)
	}

	// Check that both mermaid and d2 content are present
	if !strings.Contains(rfd.Content, "mermaid") {
		t.Error("Expected mermaid content in rendered output")
	}

	if !strings.Contains(rfd.Content, "d2-diagram") {
		t.Error("Expected d2-diagram content in rendered output")
	}

	// Basic SVG check for d2 diagram
	if !strings.Contains(rfd.Content, "<svg") {
		t.Error("Expected SVG content in rendered output")
	}

	t.Logf("Rendered content length: %d", len(rfd.Content))
}

func TestD2DiagramRendering(t *testing.T) {
	// Test with just a d2 diagram
	content := `---
title: D2 Test
authors: ["test@example.com"]
state: discussion
---

# D2 Test

` + "```d2" + `
users -> database: query
database -> users: results
` + "```" + `
`

	reader := strings.NewReader(content)
	rfd, err := RenderRFD("002", reader)
	if err != nil {
		t.Fatalf("Failed to render RFD: %v", err)
	}

	// Check d2 specific content
	if !strings.Contains(rfd.Content, "d2-diagram") {
		t.Error("Expected d2-diagram wrapper in rendered output")
	}

	if !strings.Contains(rfd.Content, "<svg") {
		t.Error("Expected SVG content for d2 diagram")
	}
}

func TestD2ErrorHandling(t *testing.T) {
	// Test with invalid d2 content
	content := `---
title: D2 Error Test
authors: ["test@example.com"]
state: discussion
---

# D2 Error Test

` + "```d2" + `
this is invalid d2 syntax $$
` + "```" + `
`

	reader := strings.NewReader(content)
	rfd, err := RenderRFD("003", reader)
	if err != nil {
		t.Fatalf("Failed to render RFD: %v", err)
	}

	// Should contain error handling
	if !strings.Contains(rfd.Content, "d2-error") {
		t.Error("Expected d2-error class for invalid syntax")
	}
}