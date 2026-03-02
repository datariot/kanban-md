package task

import (
	"testing"
)

const testSectionContent = "Result: success"

func TestParseSections_Empty(t *testing.T) {
	sections := ParseSections("")
	if len(sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(sections))
	}
}

func TestParseSections_NoSections(t *testing.T) {
	body := "Just a plain body\nwith multiple lines"
	sections := ParseSections(body)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].Name != "" {
		t.Errorf("expected empty name, got %q", sections[0].Name)
	}
	if sections[0].Body != "Just a plain body\nwith multiple lines" {
		t.Errorf("unexpected body: %q", sections[0].Body)
	}
}

func TestParseSections_MultipleSections(t *testing.T) {
	body := "## Artifact\nResult: success\nSummary: done\n\n## Decisions\n1. Use chi router\n\n## Files Modified\n- internal/api/handler.go"
	sections := ParseSections(body)

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}

	want := []struct {
		name string
		body string
	}{
		{"Artifact", "Result: success\nSummary: done"},
		{"Decisions", "1. Use chi router"},
		{"Files Modified", "- internal/api/handler.go"},
	}

	for i, w := range want {
		if sections[i].Name != w.name {
			t.Errorf("section[%d].Name = %q, want %q", i, sections[i].Name, w.name)
		}
		if sections[i].Body != w.body {
			t.Errorf("section[%d].Body = %q, want %q", i, sections[i].Body, w.body)
		}
	}
}

func TestParseSections_PreambleAndSections(t *testing.T) {
	body := "This is the preamble text.\n\n## Notes\nSome notes here"
	sections := ParseSections(body)

	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Name != "" {
		t.Errorf("preamble name = %q, want empty", sections[0].Name)
	}
	if sections[0].Body != "This is the preamble text." {
		t.Errorf("preamble body = %q", sections[0].Body)
	}
	if sections[1].Name != "Notes" {
		t.Errorf("section name = %q, want Notes", sections[1].Name)
	}
}

func TestGetSection_Found(t *testing.T) {
	body := "## Artifact\nResult: success\n\n## Decisions\n1. Use chi"
	content, ok := GetSection(body, "Artifact")
	if !ok {
		t.Fatal("expected section to be found")
	}
	if content != testSectionContent {
		t.Errorf("content = %q, want %q", content, testSectionContent)
	}
}

func TestGetSection_CaseInsensitive(t *testing.T) {
	body := "## Artifact\nResult: success"
	content, ok := GetSection(body, "artifact")
	if !ok {
		t.Fatal("expected case-insensitive match")
	}
	if content != testSectionContent {
		t.Errorf("content = %q", content)
	}
}

func TestGetSection_NotFound(t *testing.T) {
	body := "## Artifact\nResult: success"
	_, ok := GetSection(body, "Missing")
	if ok {
		t.Error("expected section not to be found")
	}
}

func TestSetSection_CreateNew(t *testing.T) {
	body := "Some preamble text"
	result := SetSection(body, "Artifact", testSectionContent)
	content, ok := GetSection(result, "Artifact")
	if !ok {
		t.Fatal("expected created section to be found")
	}
	if content != testSectionContent {
		t.Errorf("content = %q", content)
	}
	// Preamble should be preserved.
	preamble, preambleOK := GetSection(result, "")
	if !preambleOK {
		t.Error("expected preamble to be preserved")
	}
	if preamble != "Some preamble text" {
		t.Errorf("preamble = %q, want %q", preamble, "Some preamble text")
	}
}

func TestSetSection_ReplaceExisting(t *testing.T) {
	body := "## Artifact\nOld content\n\n## Decisions\n1. Use chi"
	result := SetSection(body, "Artifact", "New content")

	content, ok := GetSection(result, "Artifact")
	if !ok {
		t.Fatal("expected section to exist")
	}
	if content != "New content" {
		t.Errorf("content = %q, want %q", content, "New content")
	}

	// Other sections should be preserved.
	decisions, ok := GetSection(result, "Decisions")
	if !ok {
		t.Fatal("expected Decisions section to be preserved")
	}
	if decisions != "1. Use chi" {
		t.Errorf("Decisions = %q", decisions)
	}
}

func TestSetSection_EmptyBody(t *testing.T) {
	result := SetSection("", "Artifact", testSectionContent)
	content, ok := GetSection(result, "Artifact")
	if !ok {
		t.Fatal("expected section to be found")
	}
	if content != testSectionContent {
		t.Errorf("content = %q", content)
	}
}

func TestSetSection_CaseInsensitiveReplace(t *testing.T) {
	body := "## Artifact\nOld content"
	result := SetSection(body, "artifact", "New content")

	sections := ParseSections(result)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	// The original heading case should be preserved.
	if sections[0].Name != "Artifact" {
		t.Errorf("name = %q, want original case %q", sections[0].Name, "Artifact")
	}
	if sections[0].Body != "New content" {
		t.Errorf("body = %q", sections[0].Body)
	}
}

func TestRenderSections_Roundtrip(t *testing.T) {
	original := "## Artifact\nResult: success\n\n## Decisions\n1. Use chi"
	sections := ParseSections(original)
	rendered := renderSections(sections)

	// Re-parse and compare.
	reparsed := ParseSections(rendered)
	if len(reparsed) != len(sections) {
		t.Fatalf("roundtrip: %d sections vs %d", len(reparsed), len(sections))
	}
	for i := range sections {
		if reparsed[i].Name != sections[i].Name {
			t.Errorf("roundtrip name[%d] = %q, want %q", i, reparsed[i].Name, sections[i].Name)
		}
		if reparsed[i].Body != sections[i].Body {
			t.Errorf("roundtrip body[%d] = %q, want %q", i, reparsed[i].Body, sections[i].Body)
		}
	}
}
