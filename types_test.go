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
