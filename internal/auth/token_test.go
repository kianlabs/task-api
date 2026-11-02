package auth

import "testing"

const secret = "test-secret-key-at-least-32-characters"

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := GenerateToken(42, secret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	userID, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if userID != 42 {
		t.Errorf("expected user id 42, got %d", userID)
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	token, err := GenerateToken(1, secret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := ValidateToken(token, "another-secret-key-at-least-32-chars"); err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}

func TestValidateTokenTampered(t *testing.T) {
	token, err := GenerateToken(1, secret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	tampered := token
	if i := len(token) / 2; i > 0 && i < len(token)-1 {
		replacement := byte('a')
		if token[i] == 'a' {
			replacement = 'b'
		}
		tampered = token[:i] + string(replacement) + token[i+1:]
	}
	if _, err := ValidateToken(tampered, secret); err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestValidateTokenGarbage(t *testing.T) {
	if _, err := ValidateToken("not-a-token", secret); err == nil {
		t.Error("expected error for garbage token")
	}
}
