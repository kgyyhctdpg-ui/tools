package excel

import (
	"path/filepath"
	"testing"
)

func TestExportMapsAndReadMaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.xlsx")
	err := ExportMaps(path, "Users", []Header{
		{Title: "Name", Key: "name", Width: 16},
		{Title: "Age", Key: "age", Width: 10},
	}, []map[string]any{
		{"name": "Alice", "age": 18},
		{"name": "Bob", "age": 20},
	}, WithFreezeHeader(), WithAutoFilter())
	if err != nil {
		t.Fatalf("ExportMaps failed: %v", err)
	}

	rows, err := ReadMaps(path, "Users", 1)
	if err != nil {
		t.Fatalf("ReadMaps failed: %v", err)
	}
	if len(rows) != 2 || rows[0]["Name"] != "Alice" || rows[1]["Age"] != "20" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestExportRowsAndReadRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rows.xlsx")
	err := ExportRows(path, "", []string{"A", "B"}, [][]any{
		{"x", 1},
		{"y", 2},
	})
	if err != nil {
		t.Fatalf("ExportRows failed: %v", err)
	}

	rows, err := ReadRows(path, "")
	if err != nil {
		t.Fatalf("ReadRows failed: %v", err)
	}
	if len(rows) != 3 || rows[0][0] != "A" || rows[2][1] != "2" {
		t.Fatalf("rows = %#v", rows)
	}
}
