package email

import (
	"slices"
	"strings"

	"github.com/tavocg/go-email/blacklist"
)

// DefaultBlacklist contains the embedded disposable-domain list loaded by the
// blacklist package at startup. Reassign it if you need a different default.
var DefaultBlacklist = blacklist.DefaultDomains

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

// IsBlacklisted reports whether the receiver's domain matches any blacklist
// entry. When no custom blacklist is provided, it falls back to DefaultBlacklist.
func (v *ValidAddress) IsBlacklisted(blacklists ...[]string) bool {
	lists := blacklists
	if len(lists) == 0 {
		lists = [][]string{DefaultBlacklist}
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

// validAddress remains as a private alias to avoid breaking internal package
// references while the exported name is adopted.
type validAddress = ValidAddress
