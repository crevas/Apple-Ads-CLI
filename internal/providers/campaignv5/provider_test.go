package campaignv5

import (
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

func TestCampaignReportPayloadRequestsRowTotalsWithGrandTotals(t *testing.T) {
	payload := campaignReportPayload(appleads.CampaignReportQuery{
		AppID:       "999999999",
		From:        "2026-06-01",
		To:          "2026-06-30",
		Granularity: "DAILY",
		TimeZone:    "ORTZ",
	})

	if got := payload["returnGrandTotals"]; got != true {
		t.Fatalf("returnGrandTotals = %v, want true", got)
	}
	if got := payload["returnRowTotals"]; got != true {
		t.Fatalf("returnRowTotals = %v, want true when grand totals are requested", got)
	}
}

func TestLegacyPlanRetainsPlacementAndNormalizesMoney(t *testing.T) {
	input := appleads.CampaignCreate{Supply: "APPSTORE_PRODUCT_PAGES", Currency: "CNY", DailyBudget: "200"}
	payload := New("654").campaignPayload(input)
	if got := payload["supplySources"].([]string)[0]; got != "APPSTORE_PRODUCT_PAGES_BROWSE" {
		t.Fatal(got)
	}
	if got := payload["dailyBudgetAmount"].(appleads.Money); got.Currency != "RMB" || got.Amount != "200" {
		t.Fatal(got)
	}
	group := adGroupPayload(appleads.AdGroupCreate{Currency: "cny", Bid: "10", CPAGoal: "5"})
	if group["defaultBidAmount"].(appleads.Money).Currency != "RMB" || group["cpaGoal"].(appleads.Money).Currency != "RMB" {
		t.Fatal(group)
	}
	keyword := keywordPayload(appleads.KeywordCreate{Currency: " CNY ", Bid: "4", Text: "CNY exchange"})
	if keyword["bidAmount"].(appleads.Money).Currency != "RMB" || keyword["text"] != "CNY exchange" {
		t.Fatal(keyword)
	}
}
