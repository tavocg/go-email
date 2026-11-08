package blacklist

import (
	"strings"
	"sync"
	"testing"
)

func TestContainsEmbeddedDomains(t *testing.T) {
	for _, domain := range strings.Fields(embeddedDomains) {
		if !Contains(domain) {
			t.Fatalf("embedded domain %q missing from the lookup", domain)
		}
	}
	if Contains("allowed.invalid") {
		t.Fatal("unlisted domain matched")
	}
	if !Contains(" \t" + strings.ToUpper(DefaultDomains[0]) + "\n") {
		t.Fatal("embedded lookup did not normalize its input")
	}
}

func TestContainsCustomLists(t *testing.T) {
	if Contains(" \t", []string{"", " "}) {
		t.Fatal("blank entries matched an empty domain")
	}
	if !Contains(" BLOCKED.EXAMPLE ", []string{"allowed.example"}, []string{" blocked.example\t"}) {
		t.Fatal("custom lists did not normalize domains or check every list")
	}
	if Contains(DefaultDomains[0], nil) || Contains(DefaultDomains[0], []string{}) {
		t.Fatal("explicit empty list fell back to the embedded set")
	}
}

func TestContainsConcurrentWithSnapshotMutation(t *testing.T) {
	snapshot := DefaultDomains
	first := snapshot[0]
	defer func() {
		snapshot[0] = first
		DefaultDomains = snapshot
	}()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !Contains(first) || Contains("custom.invalid") {
					t.Error("default lookup changed with its inspection snapshot")
					return
				}
			}
		}()
	}
	for i := 0; i < 100; i++ {
		snapshot[0] = "changed.invalid"
		DefaultDomains = []string{"custom.invalid"}
	}
	wg.Wait()
}
