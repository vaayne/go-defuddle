package defuddle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// expectedMetadata matches the JSON preamble in expected .md files.
type expectedMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Site      string `json:"site"`
	Published string `json:"published"`
}

// parseExpectedFile splits a .md expected file into metadata and markdown body.
func parseExpectedFile(content string) (expectedMetadata, string, error) {
	var meta expectedMetadata

	re := regexp.MustCompile("(?s)^```json\n(.*?)\n```\n\n")
	matches := re.FindStringSubmatch(content)
	if matches == nil {
		return meta, strings.TrimSpace(content), nil
	}

	if err := json.Unmarshal([]byte(matches[1]), &meta); err != nil {
		return meta, "", fmt.Errorf("parse metadata JSON: %w", err)
	}

	body := strings.TrimSpace(content[len(matches[0]):])
	return meta, body, nil
}

// extractURLFromFixture extracts the URL from a <!-- {"url":"..."} --> comment.
func extractURLFromFixture(html string) string {
	re := regexp.MustCompile(`<!--\s*(\{[^}]*"url"\s*:[^}]*\})\s*-->`)
	matches := re.FindStringSubmatch(html)
	if matches == nil {
		return ""
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(matches[1]), &obj); err != nil {
		return ""
	}
	return obj.URL
}

// fixtureURLFromName derives a URL from the fixture filename (fallback).
func fixtureURLFromName(name string) string {
	re := regexp.MustCompile(`^[a-z]+--`)
	urlName := re.ReplaceAllString(name, "")
	return "https://" + urlName
}

// TestFixtures runs all HTML fixtures through the Go parser and compares
// metadata (title, author, site, published) against expected .md files.
// HTML content is compared against expected .html files when they exist.
func TestFixtures(t *testing.T) {
	fixturesDir := filepath.Join("defuddle", "tests", "fixtures")
	expectedDir := filepath.Join("defuddle", "tests", "expected")

	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "*.html"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found — is the defuddle submodule checked out?")
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	for _, fixturePath := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixturePath), ".html")

		t.Run(name, func(t *testing.T) {
			htmlBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			html := string(htmlBytes)

			url := extractURLFromFixture(html)
			if url == "" {
				url = fixtureURLFromName(name)
			}

			result, err := parser.Parse(html, url, nil)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Basic validation
			if result.Content == "" {
				t.Error("content is empty")
			}

			// Check expected markdown file for metadata
			expectedMdPath := filepath.Join(expectedDir, name+".md")
			expectedBytes, err := os.ReadFile(expectedMdPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("no expected file for %s", name)
					return
				}
				t.Fatalf("read expected: %v", err)
			}

			expectedMeta, _, err := parseExpectedFile(string(expectedBytes))
			if err != nil {
				t.Fatalf("parse expected file: %v", err)
			}

			if result.Title != expectedMeta.Title {
				t.Errorf("title mismatch:\n  got:  %q\n  want: %q", result.Title, expectedMeta.Title)
			}
			if result.Author != expectedMeta.Author {
				t.Errorf("author mismatch:\n  got:  %q\n  want: %q", result.Author, expectedMeta.Author)
			}
			if result.Site != expectedMeta.Site {
				t.Errorf("site mismatch:\n  got:  %q\n  want: %q", result.Site, expectedMeta.Site)
			}
			if result.Published != expectedMeta.Published {
				t.Errorf("published mismatch:\n  got:  %q\n  want: %q", result.Published, expectedMeta.Published)
			}

			// Check expected HTML file (if exists)
			expectedHtmlPath := filepath.Join(expectedDir, name+".html")
			expectedHtmlBytes, err := os.ReadFile(expectedHtmlPath)
			if err == nil {
				expectedHtml := strings.TrimSpace(string(expectedHtmlBytes))
				gotHtml := strings.TrimSpace(result.Content)
				if gotHtml != expectedHtml {
					t.Errorf("HTML content mismatch\n  got length:  %d\n  want length: %d", len(gotHtml), len(expectedHtml))
				}
			}
		})
	}
}

// TestFixtures_MarkdownNonEmpty verifies that markdown conversion produces
// non-empty output for every fixture. Exact markdown comparison is skipped
// because Go's html-to-markdown and JS's Turndown produce different output.
func TestFixtures_MarkdownNonEmpty(t *testing.T) {
	fixturesDir := filepath.Join("defuddle", "tests", "fixtures")

	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "*.html"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found — is the defuddle submodule checked out?")
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	defer parser.Close()

	for _, fixturePath := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixturePath), ".html")

		t.Run(name, func(t *testing.T) {
			htmlBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			html := string(htmlBytes)

			url := extractURLFromFixture(html)
			if url == "" {
				url = fixtureURLFromName(name)
			}

			result, err := parser.Parse(html, url, &Options{Markdown: true})
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if strings.TrimSpace(result.Markdown) == "" {
				t.Error("markdown is empty")
			}
		})
	}
}
