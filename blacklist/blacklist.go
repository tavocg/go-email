// Package blacklist loads blacklisted domains into memory
package blacklist

import (
	_ "embed"
	"strings"
)

//go:embed domains.txt
var embeddedDomains string

// DefaultDomains contains the embedded disposable-domain list parsed at
// startup from domains.txt.
var DefaultDomains []string

func init() {
	lines := strings.Split(embeddedDomains, "\n")
	DefaultDomains = make([]string, 0, len(lines))

	for _, line := range lines {
		domain := strings.TrimSpace(line)
		if domain == "" {
			continue
		}
		DefaultDomains = append(DefaultDomains, domain)
	}
}
