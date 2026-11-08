// Package blacklist loads blacklisted domains into memory.
//
// Default blocklist used here can be found at:
// https://raw.githubusercontent.com/disposable/disposable-email-domains/master/domains.txt
package blacklist

import (
	_ "embed"
	"strings"
)

//go:embed domains.txt
var embeddedDomains string

// DefaultDomains is a snapshot of the embedded disposable-domain list.
// Mutating it does not change the default lookup used by Contains.
//
// Deprecated: use Contains for lookups. Pass a custom list explicitly to
// Contains or ValidAddress.IsBlacklisted to customize filtering.
var DefaultDomains []string

var embeddedSet map[string]struct{}

func init() {
	lines := strings.Split(embeddedDomains, "\n")
	DefaultDomains = make([]string, 0, len(lines))
	embeddedSet = make(map[string]struct{}, len(lines))

	for _, line := range lines {
		domain := strings.ToLower(strings.TrimSpace(line))
		if domain == "" {
			continue
		}
		DefaultDomains = append(DefaultDomains, domain)
		embeddedSet[domain] = struct{}{}
	}
}

// Contains matches an exact domain, ignoring case and surrounding whitespace.
// With no custom lists, it uses the immutable embedded set. An explicit empty
// or nil list checks no domains and does not fall back to the embedded set.
func Contains(domain string, lists ...[]string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}
	if len(lists) == 0 {
		_, found := embeddedSet[strings.ToLower(domain)]
		return found
	}
	for _, list := range lists {
		for _, entry := range list {
			if strings.EqualFold(strings.TrimSpace(entry), domain) {
				return true
			}
		}
	}
	return false
}
