// Package email validates and normalizes email addresses for storage and
// comparison.
package email

import (
	"strings"
)

// Normalize canonicalizes an email address for storage and comparison.
//
// Apply the same normalization anywhere the address is stored, matched, or used
// for verification so those operations stay consistent.
func (v *validAddress) Normalize(email string) string {
	email = strings.ToLower(email)
	user, host, _ := strings.Cut(email, "@")
	user, _, _ = strings.Cut(user, "+")
	return user + "@" + host
}
