package onlyoffice

import (
	"testing"
	"time"
)

func TestGenerateAndVerifyToken(t *testing.T) {
	token, err := GenerateToken(map[string]interface{}{
		"document": map[string]interface{}{
			"title": "report.docx",
		},
	}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyToken(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	document, ok := claims["document"].(map[string]interface{})
	if !ok || document["title"] != "report.docx" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if _, err := VerifyToken(token, "wrong"); err == nil {
		t.Fatal("wrong secret should fail")
	}
}

func TestDocumentAccessToken(t *testing.T) {
	token, err := GenerateDocumentAccessToken("doc-1", "secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	documentID, err := VerifyDocumentAccessToken(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if documentID != "doc-1" {
		t.Fatalf("unexpected document id: %q", documentID)
	}

	expired, err := GenerateDocumentAccessToken("doc-1", "secret", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyDocumentAccessToken(expired, "secret"); err == nil {
		t.Fatal("expired token should fail")
	}
}
