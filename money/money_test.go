package money

import "testing"

func TestArithmeticAndFormat(t *testing.T) {
	sum, err := Add("0.1", "0.2")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if sum.String() != "0.3" {
		t.Fatalf("sum = %s, want 0.3", sum)
	}

	div, err := Div("1", "4")
	if err != nil {
		t.Fatalf("Div failed: %v", err)
	}
	if div.String() != "0.25" {
		t.Fatalf("div = %s, want 0.25", div)
	}

	formatted, err := Format("12.345", 2)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	if formatted != "12.35" {
		t.Fatalf("formatted = %q, want 12.35", formatted)
	}
}

func TestYuanFen(t *testing.T) {
	fen, err := YuanToFen("12.345")
	if err != nil {
		t.Fatalf("YuanToFen failed: %v", err)
	}
	if fen != 1235 {
		t.Fatalf("fen = %d, want 1235", fen)
	}
	yuan := FenToYuan(1235)
	if yuan.StringFixed(2) != "12.35" {
		t.Fatalf("yuan = %s, want 12.35", yuan.StringFixed(2))
	}
}

func TestChineseRMB(t *testing.T) {
	tests := map[string]string{
		"0":        "零元整",
		"1001":     "壹仟零壹元整",
		"1234.56":  "壹仟贰佰叁拾肆元伍角陆分",
		"10001.05": "壹万零壹元零伍分",
	}
	for input, want := range tests {
		got, err := ChineseRMB(input)
		if err != nil {
			t.Fatalf("ChineseRMB(%s) failed: %v", input, err)
		}
		if got != want {
			t.Fatalf("ChineseRMB(%s) = %q, want %q", input, got, want)
		}
	}
}
