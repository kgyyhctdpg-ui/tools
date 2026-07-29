package validator

import "testing"

func TestBasicValidators(t *testing.T) {
	if !IsMobile("13800138000") || IsMobile("12800138000") {
		t.Fatal("IsMobile returned unexpected result")
	}
	if !IsEmail("user@example.com") || IsEmail("Name <user@example.com>") {
		t.Fatal("IsEmail returned unexpected result")
	}
	if !IsURL("https://example.com/a?b=1") || IsURL("ftp://example.com") {
		t.Fatal("IsURL returned unexpected result")
	}
	if !IsIP("127.0.0.1") || !IsIPv4("127.0.0.1") || IsIPv4("2001:db8::1") {
		t.Fatal("IP validators returned unexpected result")
	}
	if !IsIPv6("2001:db8::1") {
		t.Fatal("IsIPv6 should accept IPv6")
	}
}

func TestIDCard(t *testing.T) {
	info, ok := ParseIDCard("11010519491231002X")
	if !ok {
		t.Fatal("ParseIDCard should accept known valid ID card")
	}
	if info.Birthday.Format("2006-01-02") != "1949-12-31" {
		t.Fatalf("birthday = %s, want 1949-12-31", info.Birthday.Format("2006-01-02"))
	}
	if info.Gender != "female" {
		t.Fatalf("gender = %q, want female", info.Gender)
	}
	if IsIDCard("110105194912310021") {
		t.Fatal("IsIDCard should reject invalid checksum")
	}
}

func TestUnifiedSocialCreditCode(t *testing.T) {
	if !IsUnifiedSocialCreditCode("91350211M000100Y46") {
		t.Fatal("IsUnifiedSocialCreditCode should accept valid code")
	}
	if IsUnifiedSocialCreditCode("91350211M000100Y43") {
		t.Fatal("IsUnifiedSocialCreditCode should reject invalid checksum")
	}
}

func TestBusinessValidators(t *testing.T) {
	if !IsBankCard("4111111111111111") || IsBankCard("4111111111111112") {
		t.Fatal("IsBankCard returned unexpected result")
	}
	if !IsAmount("100.01") || !IsAmount("0") || IsAmount("01.00") || IsAmount("1.001") {
		t.Fatal("IsAmount returned unexpected result")
	}
	if !IsChineseName("阿布都·热合曼") || IsChineseName("张·") {
		t.Fatal("IsChineseName returned unexpected result")
	}
	if !IsStrongPassword("Abcdef1!") || IsStrongPassword("abcdefgh") {
		t.Fatal("IsStrongPassword returned unexpected result")
	}
	if PasswordStrength("Abcdef1!") != 4 {
		t.Fatal("PasswordStrength should return 4 for mixed password")
	}
}
