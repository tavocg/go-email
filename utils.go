package email

import (
	"strings"
)

// ValidAddress is an email address that has already passed package validation.
//
// It exists so Normalize can be limited to addresses returned by the package
// parsers instead of arbitrary strings.
type ValidAddress string

// Normalize canonicalizes an email address for storage and comparison.
//
// Apply the same normalization anywhere the address is stored, matched, or used
// for verification so those operations stay consistent.
func (v *ValidAddress) Normalize(email string) string {
	email = strings.ToLower(email)
	user, host, _ := strings.Cut(email, "@")
	user, _, _ = strings.Cut(user, "+")
	return user + "@" + host
}

// validAddress remains as a private alias to avoid breaking internal package
// references while the exported name is adopted.
type validAddress = ValidAddress
