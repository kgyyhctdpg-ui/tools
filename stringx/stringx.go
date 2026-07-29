// Package stringx provides common string helpers.
package stringx

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var wordBoundaryRE = regexp.MustCompile(`[A-Za-z0-9]+`)

// IsBlank reports whether s contains only whitespace.
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// DefaultIfBlank returns fallback when s is blank.
func DefaultIfBlank(s, fallback string) string {
	if IsBlank(s) {
		return fallback
	}
	return s
}

// RuneLen returns the number of runes in s.
func RuneLen(s string) int {
	return utf8.RuneCountInString(s)
}

// Substr returns at most length runes starting from start. Negative start counts
// from the end.
func Substr(s string, start, length int) string {
	runes := []rune(s)
	if length <= 0 || len(runes) == 0 {
		return ""
	}
	if start < 0 {
		start = len(runes) + start
	}
	if start < 0 {
		start = 0
	}
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// Truncate returns s shortened to max runes. When truncated, suffix is appended
// within the max length when possible.
func Truncate(s string, max int, suffix string) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	suffixRunes := []rune(suffix)
	if len(suffixRunes) >= max {
		return string(suffixRunes[:max])
	}
	return string(runes[:max-len(suffixRunes)]) + suffix
}

// SafeByteSubstr truncates s by bytes without splitting a UTF-8 rune.
func SafeByteSubstr(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if maxBytes >= len(s) {
		return s
	}
	for maxBytes > 0 && !utf8.ValidString(s[:maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

// PadLeft pads s on the left until it reaches width runes.
func PadLeft(s string, width int, pad string) string {
	return padString(s, width, pad, true)
}

// PadRight pads s on the right until it reaches width runes.
func PadRight(s string, width int, pad string) string {
	return padString(s, width, pad, false)
}

// ZeroPadInt formats n and left-pads it with zeros to width.
func ZeroPadInt(n int64, width int) string {
	return PadLeft(fmt.Sprintf("%d", n), width, "0")
}

// Mask masks the middle of s, preserving left and right runes.
func Mask(s string, left, right int, mask string) string {
	if mask == "" {
		mask = "*"
	}
	runes := []rune(s)
	if left < 0 {
		left = 0
	}
	if right < 0 {
		right = 0
	}
	if left+right >= len(runes) {
		return strings.Repeat(mask, len(runes))
	}
	return string(runes[:left]) + strings.Repeat(mask, len(runes)-left-right) + string(runes[len(runes)-right:])
}

// MaskMobile masks a Chinese mobile number as 138****8000.
func MaskMobile(mobile string) string {
	return Mask(strings.TrimSpace(mobile), 3, 4, "*")
}

// Reverse reverses s by rune.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ContainsAny reports whether s contains any non-empty item.
func ContainsAny(s string, items ...string) bool {
	for _, item := range items {
		if item != "" && strings.Contains(s, item) {
			return true
		}
	}
	return false
}

// Unique removes duplicated strings while preserving order.
func Unique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// SplitAndTrim splits s by sep, trims spaces, and removes empty values.
func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ToSnake converts a phrase or CamelCase identifier to snake_case.
func ToSnake(s string) string {
	return strings.Join(words(s), "_")
}

// ToKebab converts a phrase or CamelCase identifier to kebab-case.
func ToKebab(s string) string {
	return strings.Join(words(s), "-")
}

// ToCamel converts a phrase or snake/kebab identifier to lower camelCase.
func ToCamel(s string) string {
	items := words(s)
	if len(items) == 0 {
		return ""
	}
	for i := 1; i < len(items); i++ {
		items[i] = upperFirst(items[i])
	}
	return strings.Join(items, "")
}

// Random returns a cryptographically secure random string using letters and
// digits.
func Random(n int) (string, error) {
	return RandomFromAlphabet(n, defaultAlphabet)
}

// RandomDigits returns a cryptographically secure numeric string.
func RandomDigits(n int) (string, error) {
	return RandomFromAlphabet(n, "0123456789")
}

// RandomFromAlphabet returns a cryptographically secure random string using the
// provided alphabet.
func RandomFromAlphabet(n int, alphabet string) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("stringx: random length cannot be negative")
	}
	if alphabet == "" {
		return "", fmt.Errorf("stringx: alphabet is empty")
	}
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[index.Int64()]
	}
	return string(out), nil
}

func padString(s string, width int, pad string, left bool) string {
	if pad == "" {
		pad = " "
	}
	length := RuneLen(s)
	if length >= width {
		return s
	}
	padding := repeatToRunes(pad, width-length)
	if left {
		return padding + s
	}
	return s + padding
}

func repeatToRunes(s string, count int) string {
	if count <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return strings.Repeat(" ", count)
	}
	out := make([]rune, 0, count)
	for len(out) < count {
		for _, r := range runes {
			out = append(out, r)
			if len(out) == count {
				break
			}
		}
	}
	return string(out)
}

func words(s string) []string {
	s = splitCamel(s)
	matches := wordBoundaryRE.FindAllString(s, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		match = strings.ToLower(match)
		if match != "" {
			result = append(result, match)
		}
	}
	return result
}

func splitCamel(s string) string {
	var out []rune
	var prev rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
			out = append(out, ' ')
		}
		out = append(out, r)
		prev = r
	}
	return string(out)
}

func upperFirst(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
