package email

import (
	"strings"
)

// validAddress implements Normalize(email string) string, it is private to
// avoid using Normalize on non-valid email addresses.
type validAddress string

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
