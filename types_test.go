package main

import (
	"encoding/json"
	"testing"
)

// ---- MarshalJSON tests ----

func TestMarshalJSON_OmitsEmptyOptionalFields(t *testing.T) {
	tag := Tag{
		Type:  "tag",
		Name:  "Foo",
		Path:  "foo.go",
		Extra: map[string]string{},
	}
	b, err := json.Marshal(tag)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pattern", "language", "kind", "line"} {
		if _, exists := m[key]; exists {
			t.Errorf("expected %q to be absent when zero/empty, but found in JSON", key)
		}
	}
}

func TestMarshalJSON_InlinesExtraFields(t *testing.T) {
	tag := Tag{
		Type: "tag",
		Name: "MyFunc",
		Path: "foo.go",
		Extra: map[string]string{
			"end":       "42",
			"signature": "(int x)",
		},
	}
	b, err := json.Marshal(tag)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["end"] != "42" {
		t.Errorf("expected Extra field 'end' to be inlined at top level, got %v", m["end"])
	}
	if m["signature"] != "(int x)" {
		t.Errorf("expected Extra field 'signature' to be inlined at top level, got %v", m["signature"])
	}
	if _, exists := m["Extra"]; exists {
		t.Error("expected 'Extra' key to not appear in JSON output")
	}
}
