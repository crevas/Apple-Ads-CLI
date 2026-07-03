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
