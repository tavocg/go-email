package email

import (
	"slices"
	"strings"

	"github.com/tavocg/go-email/blacklist"
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

// IsBlacklisted reports whether the receiver's domain matches any blacklist
// entry. When no custom blacklist is provided, it falls back to DefaultBlacklist.
func (v *validAddress) IsBlacklisted(blacklists ...[]string) bool {
	lists := blacklists
	if len(lists) == 0 {
		lists = [][]string{blacklist.DefaultDomains}
	}

	// No need to check if host part is found since this is already a valid
	// email address.
	_, domain, _ := strings.Cut(strings.ToLower(string(*v)), "@")

	for _, blacklist := range lists {
		if slices.Contains(blacklist, domain) {
			return true
		}
	}

	return false
}
