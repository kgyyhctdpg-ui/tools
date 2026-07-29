// Package validator contains framework-free validation helpers.
package validator

import (
	"net"
	"net/mail"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	mobileRE      = regexp.MustCompile(`^1[3-9]\d{9}$`)
	amountRE      = regexp.MustCompile(`^(0|[1-9]\d*)(\.\d{1,2})?$`)
	chineseNameRE = regexp.MustCompile(`^[\p{Han}]{1,15}(·[\p{Han}]{1,15})?$`)
)

// IsMobile reports whether mobile is a mainland China mobile phone number.
func IsMobile(mobile string) bool {
	return mobileRE.MatchString(strings.TrimSpace(mobile))
}

// IsEmail reports whether email is a plain email address.
func IsEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" || strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	if addr.Address != email {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	return len(parts) == 2 && strings.Contains(parts[1], ".")
}

// IsURL reports whether rawURL is an absolute HTTP(S) URL.
func IsURL(rawURL string) bool {
	parsed, err := neturl.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// IsIP reports whether value is an IPv4 or IPv6 address.
func IsIP(value string) bool {
	return net.ParseIP(strings.TrimSpace(value)) != nil
}

// IsIPv4 reports whether value is an IPv4 address.
func IsIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil
}

// IsIPv6 reports whether value is an IPv6 address.
func IsIPv6(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() == nil
}

// NormalizeIDCard returns an upper-case ID card number without surrounding
// whitespace.
func NormalizeIDCard(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// IDCardInfo contains parsed information from a valid mainland China ID card.
type IDCardInfo struct {
	Birthday   time.Time
	Gender     string
	GenderCode int
}

// IsIDCard reports whether id is a valid mainland China 18-digit ID card
// number.
func IsIDCard(id string) bool {
	_, ok := ParseIDCard(id)
	return ok
}

// ParseIDCard parses a valid mainland China 18-digit ID card number.
func ParseIDCard(id string) (IDCardInfo, bool) {
	id = NormalizeIDCard(id)
	if len(id) != 18 {
		return IDCardInfo{}, false
	}
	for i := 0; i < 17; i++ {
		if id[i] < '0' || id[i] > '9' {
			return IDCardInfo{}, false
		}
	}
	if !validIDCardChecksum(id) {
		return IDCardInfo{}, false
	}

	birthday, err := time.Parse("20060102", id[6:14])
	if err != nil || birthday.Format("20060102") != id[6:14] {
		return IDCardInfo{}, false
	}
	if birthday.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) || birthday.After(time.Now()) {
		return IDCardInfo{}, false
	}

	genderCode, err := strconv.Atoi(id[14:17])
	if err != nil {
		return IDCardInfo{}, false
	}
	gender := "female"
	if genderCode%2 == 1 {
		gender = "male"
	}
	return IDCardInfo{
		Birthday:   birthday,
		Gender:     gender,
		GenderCode: genderCode,
	}, true
}

func validIDCardChecksum(id string) bool {
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkChars := "10X98765432"

	sum := 0
	for i, weight := range weights {
		sum += int(id[i]-'0') * weight
	}
	return id[17] == checkChars[sum%11]
}

// IsUnifiedSocialCreditCode reports whether code is a valid 18-character
// Chinese unified social credit code.
func IsUnifiedSocialCreditCode(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 18 {
		return false
	}

	charset := "0123456789ABCDEFGHJKLMNPQRTUWXY"
	weights := []int{1, 3, 9, 27, 19, 26, 16, 17, 20, 29, 25, 13, 8, 24, 10, 30, 28}

	sum := 0
	for i, weight := range weights {
		index := strings.IndexByte(charset, code[i])
		if index < 0 {
			return false
		}
		sum += index * weight
	}

	checkIndex := (31 - sum%31) % 31
	return code[17] == charset[checkIndex]
}

// IsBankCard reports whether card passes a basic length and Luhn check.
func IsBankCard(card string) bool {
	card = strings.ReplaceAll(strings.TrimSpace(card), " ", "")
	if len(card) < 12 || len(card) > 19 {
		return false
	}

	sum := 0
	double := false
	for i := len(card) - 1; i >= 0; i-- {
		c := card[i]
		if c < '0' || c > '9' {
			return false
		}
		n := int(c - '0')
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}

// IsAmount reports whether value is a non-negative amount with up to 2 decimal
// places.
func IsAmount(value string) bool {
	return amountRE.MatchString(strings.TrimSpace(value))
}

// IsChineseName reports whether name looks like a Chinese name. It supports a
// single middle dot for names such as "阿布都·热合曼".
func IsChineseName(name string) bool {
	return chineseNameRE.MatchString(strings.TrimSpace(name))
}

// PasswordStrength returns a score from 0 to 4 based on length and character
// variety.
func PasswordStrength(password string) int {
	if len([]rune(password)) < 8 {
		return 0
	}

	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	score := 0
	for _, ok := range []bool{hasLower, hasUpper, hasDigit, hasSymbol} {
		if ok {
			score++
		}
	}
	return score
}

// IsStrongPassword reports whether password has at least 8 characters and at
// least three character categories.
func IsStrongPassword(password string) bool {
	return PasswordStrength(password) >= 3
}
