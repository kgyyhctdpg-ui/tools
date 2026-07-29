package jsonx

import "testing"

func TestMarshalCompactPrettyValid(t *testing.T) {
	text, err := MarshalString(map[string]any{"name": "gavin", "age": 18})
	if err != nil {
		t.Fatalf("MarshalString failed: %v", err)
	}
	if !Valid(text) {
		t.Fatalf("json should be valid: %s", text)
	}

	compact := Compact("{\n  \"a\": 1\n}")
	if compact != `{"a":1}` {
		t.Fatalf("Compact = %q, want compact json", compact)
	}

	pretty, err := Pretty(`{"a":1}`)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}
	if pretty != "{\n  \"a\": 1\n}" {
		t.Fatalf("Pretty = %q", pretty)
	}
}

func TestGetSetDelete(t *testing.T) {
	text := `{"user":{"name":"gavin"},"items":[{"id":7}]}`
	value, ok, err := Get(text, "items.0.id")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !ok || value.(float64) != 7 {
		t.Fatalf("Get = %#v/%v, want 7/true", value, ok)
	}

	updated, err := Set(text, "user.city", "shanghai")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	got, ok, err := GetString(updated, "user.city")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if !ok || got != "shanghai" {
		t.Fatalf("city = %q/%v, want shanghai/true", got, ok)
	}

	deleted, err := Delete(updated, "user.name")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok, err := Get(deleted, "user.name"); err != nil || ok {
		t.Fatalf("deleted key ok = %v err = %v, want false/nil", ok, err)
	}
}

func TestToMapAndClone(t *testing.T) {
	source := map[string]any{"a": map[string]any{"b": "c"}}
	var target map[string]any
	if err := Clone(source, &target); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}
	target["a"].(map[string]any)["b"] = "changed"
	if source["a"].(map[string]any)["b"] != "c" {
		t.Fatal("Clone should deep-copy map")
	}

	m, err := ToMap(`{"ok":true}`)
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}
	if m["ok"] != true {
		t.Fatalf("map = %#v", m)
	}
}
