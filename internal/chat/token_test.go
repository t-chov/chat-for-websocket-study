package chat

import "testing"

func TestGenerateToken(t *testing.T) {
	chatID := "1234564"
	name := "Alice"
	salt := "oAQF6zsVq7xg3sd6"

	got := GenerateToken(chatID, name, salt)
	want := "1adbbda05794ed4157cca81666f75b47"
	if got != want {
		t.Fatalf("GenerateToken() = %s, want %s", got, want)
	}

	if !ValidateToken(chatID, name, salt, want) {
		t.Fatalf("ValidateToken should accept canonical token")
	}

	if !ValidateToken(chatID, name, salt, "1ADBBDA05794ED4157CCA81666F75B47") {
		t.Fatalf("ValidateToken should be case-insensitive")
	}

	if ValidateToken(chatID, name, salt, "deadbeef") {
		t.Fatalf("ValidateToken should reject mismatched tokens")
	}
}
