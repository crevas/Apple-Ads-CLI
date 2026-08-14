package platform

import (
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

func TestCampaignReportPayloadUsesSupportedGrandTotalOption(t *testing.T) {
	payload := campaignReportPayload(appleads.CampaignReportQuery{
		AppID:       "999999999",
		From:        "2026-06-01",
		To:          "2026-06-30",
		Granularity: "DAILY",
		TimeZone:    "ORTZ",
	})
	options, ok := payload["options"].(map[string]any)
	if !ok {
		t.Fatalf("options = %#v, want map", payload["options"])
	}
	includeRows, ok := options["includeRows"].([]string)
	if !ok {
		t.Fatalf("includeRows = %#v, want []string", options["includeRows"])
	}

	if len(includeRows) != 1 || includeRows[0] != "GRAND_TOTAL" {
		t.Fatalf("includeRows = %v, want [GRAND_TOTAL]", includeRows)
	}
}
