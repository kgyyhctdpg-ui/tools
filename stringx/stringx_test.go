package stringx

import "testing"

func TestSubstrTruncateAndSafeByteSubstr(t *testing.T) {
	if got := Substr("你好world", 1, 3); got != "好wo" {
		t.Fatalf("Substr = %q, want 好wo", got)
	}
	if got := Truncate("你好世界", 3, "..."); got != "..." {
		t.Fatalf("Truncate short suffix = %q, want ...", got)
	}
	if got := Truncate("你好世界", 3, "…"); got != "你好…" {
		t.Fatalf("Truncate = %q, want 你好…", got)
	}
	if got := SafeByteSubstr("你好world", 7); got != "你好w" {
		t.Fatalf("SafeByteSubstr = %q, want 你好w", got)
	}
}

func TestPadMaskReverseAndSplit(t *testing.T) {
	if got := ZeroPadInt(12, 5); got != "00012" {
		t.Fatalf("ZeroPadInt = %q, want 00012", got)
	}
	if got := PadRight("ab", 5, "xy"); got != "abxyx" {
		t.Fatalf("PadRight = %q, want abxyx", got)
	}
	if got := MaskMobile("13800138000"); got != "138****8000" {
		t.Fatalf("MaskMobile = %q, want 138****8000", got)
	}
	if got := Reverse("你好ab"); got != "ba好你" {
		t.Fatalf("Reverse = %q, want ba好你", got)
	}
	parts := SplitAndTrim(" a, ,b ", ",")
	if len(parts) != 2 || parts[0] != "a" || parts[1] != "b" {
		t.Fatalf("SplitAndTrim = %#v", parts)
	}
}

func TestCaseAndRandom(t *testing.T) {
	if got := ToSnake("UserNameID"); got != "user_name_id" {
		t.Fatalf("ToSnake = %q, want user_name_id", got)
	}
	if got := ToKebab("user name"); got != "user-name" {
		t.Fatalf("ToKebab = %q, want user-name", got)
	}
	if got := ToCamel("user_name"); got != "userName" {
		t.Fatalf("ToCamel = %q, want userName", got)
	}
	value, err := RandomDigits(6)
	if err != nil {
		t.Fatalf("RandomDigits failed: %v", err)
	}
	if len(value) != 6 {
		t.Fatalf("RandomDigits len = %d, want 6", len(value))
	}
}

func TestUniqueAndContainsAny(t *testing.T) {
	items := Unique([]string{"a", "b", "a"})
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("Unique = %#v", items)
	}
	if !ContainsAny("hello world", "x", "world") {
		t.Fatal("ContainsAny should find world")
	}
	if !IsBlank(" \n\t") || DefaultIfBlank(" ", "x") != "x" {
		t.Fatal("blank helpers returned unexpected result")
	}
}
