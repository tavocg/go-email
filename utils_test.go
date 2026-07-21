package email

import "testing"

func TestValidAddressNormalize(t *testing.T) {
	address := ValidAddress("User.Name+tag@Example.COM")

	address.Normalize()

	if got, want := address.Address(), "user.name@example.com"; got != want {
		t.Fatalf("Normalize() address = %q, want %q", got, want)
	}
}

func TestValidAddressIsBlacklistedUsesCustomLists(t *testing.T) {
	address := ValidAddress("User@Blocked.Example")

	if !address.IsBlacklisted([]string{"blocked.example"}) {
		t.Fatal("IsBlacklisted() = false, want true")
	}
	if address.IsBlacklisted([]string{"allowed.example"}) {
		t.Fatal("IsBlacklisted() = true, want false")
	}
}
