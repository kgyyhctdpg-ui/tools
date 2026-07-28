package renderer

import "testing"

func TestOrderedListNumberingStyleUsesReportBodyOrder(t *testing.T) {
	tests := []struct {
		nestedLevel int
		format      string
		pattern     string
	}{
		{nestedLevel: 0, format: "chineseCounting", pattern: "（%1）"},
		{nestedLevel: 1, format: "decimal", pattern: "%1."},
		{nestedLevel: 2, format: "decimal", pattern: "（%1）"},
		{nestedLevel: 3, format: "decimalEnclosedCircle", pattern: "%1"},
		{nestedLevel: 4, format: "upperLetter", pattern: "%1."},
		{nestedLevel: 5, format: "lowerLetter", pattern: "%1."},
		{nestedLevel: 6, format: "decimal", pattern: "%1）"},
	}

	for _, tt := range tests {
		got := orderedListNumberingStyle(tt.nestedLevel)
		if got.format != tt.format || got.pattern != tt.pattern {
			t.Fatalf("level %d style = {%q, %q}, want {%q, %q}",
				tt.nestedLevel, got.format, got.pattern, tt.format, tt.pattern)
		}
	}
}

func TestGetNumberingLevelIsAlgorithmic(t *testing.T) {
	renderer := &DocxRenderer{}
	tests := []struct {
		name        string
		nestedLevel int
		num         int
		want        string
	}{
		{name: "top level Chinese", nestedLevel: 0, num: 26, want: "（二十六）"},
		{name: "decimal dot", nestedLevel: 1, num: 102, want: "102."},
		{name: "decimal parentheses", nestedLevel: 2, num: 38, want: "（38）"},
		{name: "circled unicode", nestedLevel: 3, num: 25, want: "㉕"},
		{name: "circled fallback", nestedLevel: 3, num: 51, want: "（51）"},
		{name: "upper alpha after Z", nestedLevel: 4, num: 27, want: "AA."},
		{name: "lower alpha after z", nestedLevel: 5, num: 28, want: "ab."},
		{name: "decimal right paren", nestedLevel: 6, num: 38, want: "38）"},
		{name: "levels keep cycling", nestedLevel: 7, num: 11, want: "（十一）"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderer.getNumberingLevel(tt.nestedLevel, tt.num); got != tt.want {
				t.Fatalf("getNumberingLevel(%d, %d) = %q, want %q", tt.nestedLevel, tt.num, got, tt.want)
			}
		})
	}
}
