package integration

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLegacyLiteratureHashAndUserText(t *testing.T) {
	var fixture struct {
		Original     string            `json:"original"`
		Metadata     Object            `json:"metadata"`
		Sections     map[string]string `json:"sections"`
		Hash         string            `json:"hash"`
		ExpectedHash string            `json:"expected_hash"`
		ExpectedBody string            `json:"expected_body"`
	}
	data, err := os.ReadFile("testdata/literature-legacy.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	patched, sections, err := LiteratureNote(&fixture.Original, fixture.Metadata, fixture.Sections, fixture.Hash)
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := SplitFrontmatter(patched)
	if err != nil {
		t.Fatal(err)
	}
	expected := fixture.ExpectedBody
	if strings.TrimSpace(body) != strings.TrimSpace(expected) {
		t.Fatalf("legacy body changed unexpectedly:\n%s\nexpected:\n%s", body, expected)
	}
	raw, err := BlockText(sections)
	if err != nil || Digest(raw) != fixture.ExpectedHash {
		t.Fatalf("hash changed: %q %v", raw, err)
	}
	edited := strings.Replace(fixture.Original, "Síntese original.", "Minha edição manual.", 1)
	if _, _, err = LiteratureNote(&edited, fixture.Metadata, fixture.Sections, fixture.Hash); err == nil {
		t.Fatal("overwrote edited imported section")
	}
}
