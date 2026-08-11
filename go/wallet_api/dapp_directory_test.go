package main

import "testing"

func TestDAppDirectoryListAndFilter(t *testing.T) {
	all := listDApps("", "")
	if len(all) < 10 {
		t.Fatalf("directory has %d dapps, want >= 10", len(all))
	}
	// Every entry must have a non-empty id, name, url, category, and at least one chain.
	for _, d := range all {
		if d.ID == "" || d.Name == "" || d.URL == "" || d.Category == "" {
			t.Fatalf("incomplete dapp entry: %+v", d)
		}
		if len(d.Chains) == 0 {
			t.Fatalf("dapp %q has no chains", d.ID)
		}
	}

	defi := listDApps("defi", "")
	if len(defi) == 0 {
		t.Fatal("expected defi dapps, got 0")
	}
	for _, d := range defi {
		if d.Category != "defi" {
			t.Fatalf("filter returned non-defi dapp: %+v", d)
		}
	}

	ethOnly := listDApps("", "ethereum")
	if len(ethOnly) == 0 {
		t.Fatal("expected ethereum dapps, got 0")
	}
	for _, d := range ethOnly {
		found := false
		for _, c := range d.Chains {
			if c == "ethereum" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ethereum-filter returned dapp without ethereum chain: %+v", d)
		}
	}

	solanaOnly := listDApps("", "solana")
	for _, d := range solanaOnly {
		found := false
		for _, c := range d.Chains {
			if c == "solana" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("solana-filter returned dapp without solana chain: %+v", d)
		}
	}
}

func TestDAppGetByID(t *testing.T) {
	u := getDApp("uniswap")
	if u == nil {
		t.Fatal("uniswap not found")
	}
	if u.URL != "https://app.uniswap.org" {
		t.Fatalf("uniswap url = %q", u.URL)
	}
	if !u.Verified {
		t.Fatal("uniswap should be verified")
	}
	if getDApp("does-not-exist") != nil {
		t.Fatal("expected nil for unknown dapp id")
	}
}

func TestDAppCategories(t *testing.T) {
	cats := dAppCategories()
	if len(cats) < 2 {
		t.Fatalf("expected >= 2 categories, got %d", len(cats))
	}
	all := cats[0]
	if all["id"] != "all" {
		t.Fatalf("first category id = %v, want 'all'", all["id"])
	}
	if all["count"].(int) != len(dappDirectory) {
		t.Fatalf("all count = %v, want %d", all["count"], len(dappDirectory))
	}
}
