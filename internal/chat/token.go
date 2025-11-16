package chat

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// GenerateToken builds the md5 hash for a given chatID, participant name, and salt.
func GenerateToken(chatID, name, salt string) string {
	normalized := chatID + strings.ToLower(strings.TrimSpace(name)) + salt
	sum := md5.Sum([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// ValidateToken ensures the provided token matches the expected md5 hash.
func ValidateToken(chatID, name, salt, token string) bool {
	expected := GenerateToken(chatID, name, salt)
	return strings.EqualFold(expected, token)
}
