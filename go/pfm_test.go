package paperwork

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Fixtures come from the existing Python parser, not the Go implementation.
// Keep IDs, coordinates and review entries compatible with stored proposals.
func TestPFMPythonCompatibility(t *testing.T) {
	raw, err := os.ReadFile("testdata/pfm-python.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name     string
		Pages    []string
		Context  *string
		Expected json.RawMessage
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := Parse(c.Pages, "scan", c.Context)
			raw, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(c.Expected, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Go: %s\nPython: %s", raw, c.Expected)
			}
		})
	}
}
