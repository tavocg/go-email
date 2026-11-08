package email

import (
	"strings"

	"github.com/tavocg/go-email/blacklist"
)

// ValidAddress is an email address that has already passed package validation.
//
// It exists so Normalize can be limited to addresses returned by the package
// parsers instead of arbitrary strings.
type ValidAddress string

// NormalizeOption configures optional normalization behavior.
type NormalizeOption func(*normalizeConfig)

type normalizeConfig struct {
	stripPlusTag bool
}

// StripPlusTag removes everything after the first "+" in the local part during
// normalization.
func StripPlusTag() NormalizeOption {
	return func(config *normalizeConfig) {
		config.stripPlusTag = true
	}
}

func (v *ValidAddress) Address() string {
	return string(*v)
}

// Normalize canonicalizes the receiver for storage and comparison.
//
// Apply the same normalization anywhere the address is stored, matched, or used
// for verification so those operations stay consistent.
func (v *ValidAddress) Normalize(options ...NormalizeOption) {
	config := normalizeConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	email := strings.ToLower(v.Address())
	user, host, _ := strings.Cut(email, "@")
	if config.stripPlusTag {
		user, _, _ = strings.Cut(user, "+")
	}
	*v = ValidAddress(user + "@" + host)
}

// IsBlacklisted reports whether the receiver's domain matches any blacklist
// entry. With no custom lists, it uses the immutable embedded list. Matching
// ignores case and surrounding whitespace; subdomains require their own entry.
func (v *ValidAddress) IsBlacklisted(blacklists ...[]string) bool {
	_, domain, _ := strings.Cut(v.Address(), "@")
	return blacklist.Contains(domain, blacklists...)
}
