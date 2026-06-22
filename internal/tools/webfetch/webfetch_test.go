package webfetch

import "testing"

func TestNormalizeURLAddsHTTPSForBareDomain(t *testing.T) {
	if got := normalizeURL("flxw.dev"); got != "https://flxw.dev" {
		t.Fatalf("expected https://flxw.dev, got %q", got)
	}
}

func TestNormalizeURLKeepsExistingScheme(t *testing.T) {
	for _, raw := range []string{"https://flxw.dev", "http://flxw.dev"} {
		if got := normalizeURL(raw); got != raw {
			t.Fatalf("expected %q unchanged, got %q", raw, got)
		}
	}
}
