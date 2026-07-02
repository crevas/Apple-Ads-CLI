package cli

import (
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

func TestParsePlanCreateBusinessOptions(t *testing.T) {
	input, err := parsePlanCreate([]string{
		"--name", "AwayFinder UK Category",
		"--app-id", "999999999",
		"--country", "UK",
		"--daily-budget", "300",
		"--currency", "usd",
		"--adgroup", "AwayFinder UK Keywords",
		"--bid", "2.00",
		"--cpa-goal", "12.00",
		"--exact-keywords", "flight booking,cheap flights",
		"--broad-keywords", "travel app",
		"--negative-exact", "jobs,wallpaper",
		"--campaign-negative-broad", "free games",
		"--creative", "product-page:pp_123",
		"--correlation-id", "trace-1",
	})
	if err != nil {
		t.Fatalf("parsePlanCreate returned error: %v", err)
	}

	normalized, err := appleads.NormalizePlan(input)
	if err != nil {
		t.Fatalf("NormalizePlan returned error: %v", err)
	}

	if got := normalized.Countries; len(got) != 1 || got[0] != "GB" {
		t.Fatalf("countries = %v, want [GB]", got)
	}
	if normalized.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", normalized.Currency)
	}
	if normalized.CPAGoal != "12.00" {
		t.Fatalf("cpa goal = %q, want 12.00", normalized.CPAGoal)
	}
	if got := len(normalized.Keywords); got != 3 {
		t.Fatalf("keyword count = %d, want 3", got)
	}
	if got := len(normalized.AdGroupNegativeKeywords); got != 2 {
		t.Fatalf("ad group negative count = %d, want 2", got)
	}
	if got := len(normalized.CampaignNegativeKeywords); got != 1 {
		t.Fatalf("campaign negative count = %d, want 1", got)
	}
	if normalized.Creative.Kind != "product-page" || normalized.Creative.ProductPageID != "pp_123" {
		t.Fatalf("creative = %#v, want product-page pp_123", normalized.Creative)
	}
	if normalized.Execute {
		t.Fatal("Execute = true, want false by default")
	}
}
