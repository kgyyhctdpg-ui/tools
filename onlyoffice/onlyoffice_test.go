package onlyoffice

import (
	"testing"
	"time"
)

func TestGenerateDocumentAccessToken(t *testing.T) {
	token, err := GenerateDocumentAccessToken("1234567890", "test", 2*time.Hour)
	if err != nil {
		t.Errorf("GenerateDocumentAccessToken error: %v", err)
	}
	t.Logf("token: %s", token)
}

func TestVerifyDocumentAccessToken(t *testing.T) {
	token := "test"
	documentId, err := VerifyDocumentAccessToken(token, "test")
	if err != nil {
		t.Errorf("VerifyDocumentAccessToken error: %v", err)
	}
	t.Logf("documentId: %s, realFileUrl: %s", documentId)
}
