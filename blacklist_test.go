package email

import (
	"testing"

	"github.com/tavocg/go-email/blacklist"
)

func TestIsBlacklistedNormalizesCustomDomains(t *testing.T) {
	address := ValidAddress("User@Blocked.Example")
	if !address.IsBlacklisted([]string{" BLOCKED.Example \t"}) {
		t.Fatal("custom domain matching did not ignore case and surrounding whitespace")
	}
}

func TestIsBlacklistedPreservesCustomListSemantics(t *testing.T) {
	address := ValidAddress("User@Blocked.Example")
	if !address.IsBlacklisted([]string{"allowed.example"}, []string{"blocked.example"}) {
		t.Fatal("did not search every custom list")
	}
	list := []string{"allowed.example"}
	if address.IsBlacklisted(list) {
		t.Fatal("unlisted domain blocked")
	}
	list[0] = "blocked.example"
	if !address.IsBlacklisted(list) {
		t.Fatal("explicit custom list mutation was ignored")
	}
	address = ValidAddress("user@" + blacklist.DefaultDomains[0])
	if !address.IsBlacklisted() || address.IsBlacklisted(nil) || address.IsBlacklisted([]string{}) {
		t.Fatal("an explicit empty list incorrectly fell back to the embedded list")
	}
}

func TestIsBlacklistedMatchesExactDomain(t *testing.T) {
	for _, domain := range []string{"sub.blocked.example", "notblocked.example", "example"} {
		address := ValidAddress("user@" + domain)
		if address.IsBlacklisted([]string{"blocked.example"}) {
			t.Fatalf("nonmatching domain %q blocked", domain)
		}
	}
}

func TestIsBlacklistedDefaultIgnoresSnapshotMutation(t *testing.T) {
	first := blacklist.DefaultDomains[0]
	snapshot := blacklist.DefaultDomains
	defer func() {
		snapshot[0] = first
		blacklist.DefaultDomains = snapshot
	}()
	blacklist.DefaultDomains[0] = "changed.invalid"
	blacklist.DefaultDomains = []string{"custom.invalid"}
	blocked, allowed := ValidAddress("user@"+first), ValidAddress("user@custom.invalid")
	if !blocked.IsBlacklisted() || allowed.IsBlacklisted() {
		t.Fatal("mutating the inspection snapshot changed default filtering")
	}
}

var benchmarkBlocked bool

func BenchmarkIsBlacklisted(b *testing.B) {
	for _, tt := range []struct{ name, domain string }{
		{"present", blacklist.DefaultDomains[len(blacklist.DefaultDomains)-1]},
		{"absent", "allowed.invalid"},
	} {
		b.Run(tt.name, func(b *testing.B) {
			address := ValidAddress("user@" + tt.domain)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkBlocked = address.IsBlacklisted()
			}
		})
	}
}
