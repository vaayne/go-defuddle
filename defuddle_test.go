package defuddle

import (
	"strings"
	"testing"
)

func TestParse_BasicHTML(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	html := `<html>
<head><title>Test Article</title><meta name="author" content="Alice"></head>
<body>
<article>
<h1>Test Article</h1>
<p>First paragraph of the test article with enough content to be extracted.</p>
<h2>Section Two</h2>
<p>Second paragraph with additional content for proper word counting.</p>
</article>
<footer>Copyright 2025</footer>
</body>
</html>`

	result, err := parser.Parse(html, "https://example.com/test", nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if result.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Article")
	}
	if result.Author != "Alice" {
		t.Errorf("Author = %q, want %q", result.Author, "Alice")
	}
	if result.WordCount == 0 {
		t.Error("WordCount = 0, want > 0")
	}
	if result.Content == "" {
		t.Error("Content is empty")
	}
}

func TestParse_Markdown(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	html := `<html>
<head><title>Markdown Test</title></head>
<body>
<article>
<h1>Markdown Test</h1>
<p>This article has <strong>bold</strong> and <em>italic</em> text.</p>
<h2>Code Example</h2>
<pre><code>fmt.Println("hello")</code></pre>
<p>End of article with enough words to ensure extraction works properly here.</p>
</article>
</body>
</html>`

	result, err := parser.Parse(html, "https://example.com/md-test", &Options{Markdown: true})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if result.Markdown == "" {
		t.Fatal("Markdown is empty")
	}
	if !strings.Contains(result.Markdown, "**bold**") {
		t.Errorf("Markdown missing bold: %s", result.Markdown)
	}
	if !strings.Contains(result.Markdown, "## Code Example") {
		t.Errorf("Markdown missing heading: %s", result.Markdown)
	}
}

func TestParse_EmptyURL(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	html := `<html><head><title>No URL</title></head>
<body><article><h1>No URL</h1><p>Content without a page URL should still extract properly.</p></article></body></html>`

	result, err := parser.Parse(html, "", nil)
	if err != nil {
		t.Fatalf("Parse with empty URL: %v", err)
	}
	if result.Title != "No URL" {
		t.Errorf("Title = %q, want %q", result.Title, "No URL")
	}
}

func TestParse_MetaTags(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	html := `<html>
<head>
<title>Meta Test</title>
<meta name="description" content="A test description">
<meta property="og:image" content="https://example.com/image.jpg">
</head>
<body><article><h1>Meta Test</h1><p>Article content for the meta tag test.</p></article></body>
</html>`

	result, err := parser.Parse(html, "https://example.com/meta", nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if result.Description != "A test description" {
		t.Errorf("Description = %q, want %q", result.Description, "A test description")
	}
	if len(result.MetaTags) == 0 {
		t.Error("MetaTags is empty")
	}
}

func TestParser_Reuse(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	for i := range 3 {
		html := `<html><head><title>Reuse Test</title></head>
<body><article><h1>Reuse</h1><p>Parse call number for reuse testing of the parser instance.</p></article></body></html>`

		result, err := parser.Parse(html, "https://example.com/reuse", nil)
		if err != nil {
			t.Fatalf("Parse iteration %d: %v", i, err)
		}
		if result.Title != "Reuse Test" {
			t.Errorf("iteration %d: Title = %q", i, result.Title)
		}
	}
}
