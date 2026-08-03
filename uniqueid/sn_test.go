package uniqueid

import (
	"strings"
	"testing"
)

func TestGenSnFormat(t *testing.T) {
	sn := GenSn("ORD")
	if !strings.HasPrefix(sn, "ORD") {
		t.Fatalf("unexpected prefix: %s", sn)
	}
	if len(sn) != len("ORD")+14+8 {
		t.Fatalf("unexpected length: %d", len(sn))
	}
	for _, char := range sn[len("ORD"):] {
		if char < '0' || char > '9' {
			t.Fatalf("sn contains non-digit %q: %s", char, sn)
		}
	}
}
