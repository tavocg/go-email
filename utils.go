package email

import (
	"slices"
	"strings"

	"github.com/tavocg/go-email/blacklist"
)

// ValidAddress is an email address that has already passed package validation.
//
// It exists so Normalize can be limited to addresses returned by the package
// parsers instead of arbitrary strings.
type ValidAddress string

func (v *ValidAddress) Address() string {
	return string(*v)
}

// Normalize canonicalizes the receiver for storage and comparison.
//
// Apply the same normalization anywhere the address is stored, matched, or used
// for verification so those operations stay consistent.
func (v *ValidAddress) Normalize() {
	email := strings.ToLower(v.Address())
	user, host, _ := strings.Cut(email, "@")
	user, _, _ = strings.Cut(user, "+")
	*v = ValidAddress(user + "@" + host)
}

// IsBlacklisted reports whether the receiver's domain matches any blacklist
// entry. When no custom blacklist is provided, it falls back to DefaultBlacklist.
func (v *ValidAddress) IsBlacklisted(blacklists ...[]string) bool {
	lists := blacklists
	if len(lists) == 0 {
		lists = [][]string{blacklist.DefaultDomains}
	}

	// No need to check if host part is found since this is already a valid
	// email address.
	_, domain, _ := strings.Cut(strings.ToLower(v.Address()), "@")

	for _, blacklist := range lists {
		if slices.Contains(blacklist, domain) {
			return true
		}
	}

	return false
}
