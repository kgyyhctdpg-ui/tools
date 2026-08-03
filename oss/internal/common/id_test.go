package common

import "testing"

func TestGenSnFormat(t *testing.T) {
	sn := GenSn("OSS")
	if len(sn) != len("OSS")+14+8 {
		t.Fatalf("unexpected sn length: %q", sn)
	}
	if sn[:3] != "OSS" {
		t.Fatalf("unexpected sn prefix: %q", sn)
	}
	if !allDigits(sn[3:]) {
		t.Fatalf("sn suffix should only contain digits: %q", sn)
	}
}

func TestGenNumericID(t *testing.T) {
	id1 := GenNumericID()
	id2 := GenNumericID()
	if id1 == "" || id2 == "" {
		t.Fatal("numeric id should not be empty")
	}
	if id1 == id2 {
		t.Fatalf("numeric ids should be unique: %q", id1)
	}
	if !allDigits(id1) || !allDigits(id2) {
		t.Fatalf("numeric ids should only contain digits: %q %q", id1, id2)
	}
}

func allDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
