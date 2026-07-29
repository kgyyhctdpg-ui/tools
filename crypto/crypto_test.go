package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestHashAndHMAC(t *testing.T) {
	if MD5("hello") != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatal("MD5 returned unexpected value")
	}
	if SHA256("hello") != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatal("SHA256 returned unexpected value")
	}
	if HMACSHA256("message", "secret") != "8b5f48702995c1598c573db1e21866a9b825d4a794d169d7060a03605796360b" {
		t.Fatal("HMACSHA256 returned unexpected value")
	}
}

func TestRandom(t *testing.T) {
	value, err := RandomString(16)
	if err != nil {
		t.Fatalf("RandomString failed: %v", err)
	}
	if len(value) != 16 {
		t.Fatalf("len(RandomString) = %d, want 16", len(value))
	}
	digits, err := RandomDigits(8)
	if err != nil {
		t.Fatalf("RandomDigits failed: %v", err)
	}
	if len(digits) != 8 {
		t.Fatalf("len(RandomDigits) = %d, want 8", len(digits))
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			t.Fatalf("RandomDigits contains non-digit %q", digit)
		}
	}
	token, err := RandomToken(16)
	if err != nil {
		t.Fatalf("RandomToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("RandomToken returned empty string")
	}
}

func TestAESGCMAndCBC(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	encrypted, err := AESGCMEncryptString(key, "secret")
	if err != nil {
		t.Fatalf("AESGCMEncryptString failed: %v", err)
	}
	decrypted, err := AESGCMDecryptString(key, encrypted)
	if err != nil {
		t.Fatalf("AESGCMDecryptString failed: %v", err)
	}
	if decrypted != "secret" {
		t.Fatalf("decrypted = %q, want secret", decrypted)
	}

	cbc, err := AESCBCEncrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("AESCBCEncrypt failed: %v", err)
	}
	plain, err := AESCBCDecrypt(key, cbc)
	if err != nil {
		t.Fatalf("AESCBCDecrypt failed: %v", err)
	}
	if string(plain) != "secret" {
		t.Fatalf("plain = %q, want secret", string(plain))
	}
}

func TestRSASignVerify(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey failed: %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	})

	signature, err := RSASignSHA256(privatePEM, []byte("payload"))
	if err != nil {
		t.Fatalf("RSASignSHA256 failed: %v", err)
	}
	if err := RSAVerifySHA256(publicPEM, []byte("payload"), signature); err != nil {
		t.Fatalf("RSAVerifySHA256 failed: %v", err)
	}
	if err := RSAVerifySHA256(publicPEM, []byte("other"), signature); err == nil {
		t.Fatal("RSAVerifySHA256 should reject changed payload")
	}
}
