// Package crypto provides common hashing, signing, encryption and secure random
// helpers.
package crypto

import (
	"bytes"
	stdcrypto "crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
)

const randomAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// MD5 returns a hex-encoded MD5 digest.
func MD5(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

// SHA256 returns a hex-encoded SHA-256 digest.
func SHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// HMACSHA256 returns a hex-encoded HMAC-SHA256 signature.
func HMACSHA256(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// RandomBytes returns n cryptographically secure random bytes.
func RandomBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("crypto: random byte length cannot be negative")
	}
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}

// RandomString returns a cryptographically secure random string using letters
// and digits.
func RandomString(n int) (string, error) {
	return RandomStringFromAlphabet(n, randomAlphabet)
}

// RandomDigits returns a cryptographically secure numeric string.
func RandomDigits(n int) (string, error) {
	return RandomStringFromAlphabet(n, "0123456789")
}

// RandomStringFromAlphabet returns a cryptographically secure random string
// using the provided alphabet.
func RandomStringFromAlphabet(n int, alphabet string) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("crypto: random string length cannot be negative")
	}
	if alphabet == "" {
		return "", fmt.Errorf("crypto: alphabet is empty")
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

// RandomToken returns a URL-safe random token. byteLen controls the entropy
// bytes before base64 encoding.
func RandomToken(byteLen int) (string, error) {
	data, err := RandomBytes(byteLen)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// AESGCMEncrypt encrypts plaintext with AES-GCM. The returned bytes are
// nonce+ciphertext.
func AESGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := RandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// AESGCMDecrypt decrypts bytes produced by AESGCMEncrypt.
func AESGCMDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// AESGCMEncryptString encrypts plaintext and returns base64 URL-safe data.
func AESGCMEncryptString(key []byte, plaintext string) (string, error) {
	data, err := AESGCMEncrypt(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// AESGCMDecryptString decrypts base64 URL-safe data produced by
// AESGCMEncryptString.
func AESGCMDecryptString(key []byte, data string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	plaintext, err := AESGCMDecrypt(key, raw)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// AESCBCEncrypt encrypts plaintext with AES-CBC and PKCS#7 padding. The
// returned bytes are iv+ciphertext.
func AESCBCEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv, err := RandomBytes(block.BlockSize())
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, block.BlockSize())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	return append(iv, ciphertext...), nil
}

// AESCBCDecrypt decrypts bytes produced by AESCBCEncrypt.
func AESCBCDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(data) < blockSize || (len(data)-blockSize)%blockSize != 0 {
		return nil, fmt.Errorf("crypto: invalid CBC ciphertext length")
	}
	iv := data[:blockSize]
	ciphertext := data[blockSize:]
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return pkcs7Unpad(plaintext, blockSize)
}

// RSASignSHA256 signs data with an RSA private key in PEM format.
func RSASignSHA256(privateKeyPEM, data []byte) ([]byte, error) {
	privateKey, err := ParseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, privateKey, stdcrypto.SHA256, sum[:])
}

// RSAVerifySHA256 verifies a SHA-256 RSA signature with a public key in PEM
// format.
func RSAVerifySHA256(publicKeyPEM, data, signature []byte) error {
	publicKey, err := ParseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(publicKey, stdcrypto.SHA256, sum[:], signature)
}

// ParseRSAPrivateKey parses PKCS#1 or PKCS#8 RSA private keys.
func ParseRSAPrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("crypto: invalid private key PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("crypto: private key is not RSA")
	}
	return rsaKey, nil
}

// ParseRSAPublicKey parses PKIX or PKCS#1 RSA public keys.
func ParseRSAPublicKey(publicKeyPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("crypto: invalid public key PEM")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("crypto: public key is not RSA")
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("crypto: invalid PKCS7 data")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, fmt.Errorf("crypto: invalid PKCS7 padding")
	}
	for _, value := range data[len(data)-padding:] {
		if int(value) != padding {
			return nil, fmt.Errorf("crypto: invalid PKCS7 padding")
		}
	}
	return data[:len(data)-padding], nil
}
