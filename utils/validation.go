package utils

import (
	"net/mail"
	"strings"
)

func ValidName(name string) bool {
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func NameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ValidMail(value string, max int, allowEmpty bool) bool {
	if value == "" {
		return allowEmpty
	}
	if len(value) > max {
		return false
	}
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value
}
