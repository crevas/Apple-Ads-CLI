package platform

import (
	"encoding/json"
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

type recordingContext struct {
	method string
	path   string
	body   any
}

func (c *recordingContext) Do(method string, path string, body any) (appleads.RawResponse, error) {
	c.method = method
	c.path = path
	c.body = body
	return appleads.RawResponse{"result": []any{}}, nil
}

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

func TestKeywordRecommendationsUsePlatformOpportunityContract(t *testing.T) {
	ctx := &recordingContext{}
	_, err := New().QueryRecommendations(ctx, appleads.RecommendationQuery{
		AppID: "123456", Type: "KEYWORD", State: "AVAILABLE", Limit: 20,
	})
	if err != nil {
		t.Fatalf("QueryRecommendations returned error: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/recommendations/keywords/query" {
		t.Fatalf("request = %s %s, want POST /recommendations/keywords/query", ctx.method, ctx.path)
	}
	assertJSONEqual(t, ctx.body, map[string]any{
		"filters": []map[string]any{
			{"field": "promotedObjectId", "operator": "EQUALS", "value": []string{"123456"}},
			{"field": "promotedObjectType", "operator": "EQUALS", "value": []string{"APPSTORE_APP"}},
			{"field": "state", "operator": "EQUALS", "value": []string{"AVAILABLE"}},
		},
		"pagination": map[string]any{"offset": 0, "pageSize": 20},
	})
}

func TestSuggestionsCoverKeywordsAndTargetCPA(t *testing.T) {
	ctx := &recordingContext{}
	_, err := New().QuerySuggestions(ctx, appleads.SuggestionQuery{
		AppID: "987654", Type: "KEYWORD", Terms: []string{"task manager"}, Countries: []string{"US", "GB"}, Limit: 25,
	})
	if err != nil {
		t.Fatalf("QuerySuggestions returned error: %v", err)
	}
	if ctx.path != "/suggestions/keywords/query" {
		t.Fatalf("path = %q, want keyword suggestion path", ctx.path)
	}
	payload := ctx.body.(map[string]any)
	filters := payload["filters"].([]map[string]any)
	if len(filters) != 4 {
		t.Fatalf("filters = %#v, want four filters", filters)
	}
	assertJSONEqual(t, payload["pagination"], map[string]any{"offset": 0, "pageSize": 25})

	ctx = &recordingContext{}
	_, err = New().QuerySuggestions(ctx, appleads.SuggestionQuery{
		AppID: "987654", Type: "TARGET_CPA", Terms: []string{"ignored"}, Countries: []string{"US"},
	})
	if err != nil {
		t.Fatalf("QuerySuggestions target CPA returned error: %v", err)
	}
	if ctx.path != "/suggestions/target-cpas/query" {
		t.Fatalf("path = %q, want target CPA path", ctx.path)
	}
	payload = ctx.body.(map[string]any)
	if _, ok := payload["pagination"]; ok {
		t.Fatalf("target CPA suggestion payload must omit pagination: %#v", payload)
	}
	if got := len(payload["filters"].([]map[string]any)); got != 2 {
		t.Fatalf("target CPA filters = %d, want promoted-object filters only", got)
	}
}

func TestSearchTermPopularityNormalizesGenre(t *testing.T) {
	ctx := &recordingContext{}
	_, err := New().QuerySearchTermPopularity(ctx, appleads.SearchTermPopularityQuery{
		Country: "US", Genre: "Photo & Video", From: "2026-08-02", To: "2026-08-08", Granularity: "WEEKLY_SUN_SAT", Limit: 50,
	})
	if err != nil {
		t.Fatalf("QuerySearchTermPopularity returned error: %v", err)
	}
	if ctx.path != "/insights/apps/search-term-popularity/query" {
		t.Fatalf("path = %q, want search-term popularity path", ctx.path)
	}
	payload := ctx.body.(map[string]any)
	filters := payload["filters"].([]map[string]any)
	if filters[1]["value"] != "PHOTO_VIDEO" {
		t.Fatalf("genre = %v, want PHOTO_VIDEO", filters[1]["value"])
	}
	assertJSONEqual(t, payload["timeRange"], map[string]any{
		"start": "2026-08-02", "end": "2026-08-08", "granularity": "WEEKLY_SUN_SAT",
	})
	if _, ok := payload["sorting"]; ok {
		t.Fatalf("search-term popularity sorting is not accepted by the live v1 contract: %#v", payload["sorting"])
	}
}

func TestImpressionShareUsesAppMarketAndSearchTermFilters(t *testing.T) {
	ctx := &recordingContext{}
	_, err := New().QueryImpressionShare(ctx, appleads.ImpressionShareQuery{
		AppID: "123456", From: "2026-08-01", To: "2026-08-07", Granularity: "DAILY",
		Countries: []string{"US"}, SearchTerms: []string{"task manager"}, Limit: 50,
	})
	if err != nil {
		t.Fatalf("QueryImpressionShare returned error: %v", err)
	}
	if ctx.path != "/insights/apps/impression-share/query" {
		t.Fatalf("path = %q, want impression-share path", ctx.path)
	}
	payload := ctx.body.(map[string]any)
	assertJSONEqual(t, payload["filters"], []map[string]any{
		{"field": "promotedObjectId", "operator": "IN", "value": []string{"123456"}},
		{"field": "countryOrRegion", "operator": "IN", "value": []string{"US"}},
		{"field": "searchTerm", "operator": "IN", "value": []string{"task manager"}},
	})
	assertJSONEqual(t, payload["timeRange"], map[string]any{
		"start": "2026-08-01", "end": "2026-08-07", "timeZone": "UTC", "granularity": "DAILY",
	})
}

func TestChangeHistoryQueryAndDetailContracts(t *testing.T) {
	ctx := &recordingContext{}
	_, err := New().QueryChangeHistory(ctx, appleads.ChangeHistoryQuery{
		From: "2026-08-01", To: "2026-08-14", EntityTypes: []string{"Campaign", "Keyword"}, EventTypes: []string{"UPDATE"}, Limit: 25,
	})
	if err != nil {
		t.Fatalf("QueryChangeHistory returned error: %v", err)
	}
	if ctx.path != "/change-history/query" {
		t.Fatalf("path = %q, want change-history query path", ctx.path)
	}
	payload := ctx.body.(map[string]any)
	assertJSONEqual(t, payload["options"], map[string]any{"needTotals": "true"})
	if got := len(payload["filters"].([]map[string]any)); got != 3 {
		t.Fatalf("filters = %d, want event time, entity type, and event type", got)
	}

	_, err = New().GetChangeHistoryDetail(ctx, "Campaign/123 txn")
	if err != nil {
		t.Fatalf("GetChangeHistoryDetail returned error: %v", err)
	}
	if ctx.method != "GET" || ctx.path != "/change-history/Campaign%2F123%20txn" || ctx.body != nil {
		t.Fatalf("detail request = %s %s %#v", ctx.method, ctx.path, ctx.body)
	}
}

func assertJSONEqual(t *testing.T, actual any, expected any) {
	t.Helper()
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual: %v", err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("marshal expected: %v", err)
	}
	if string(actualJSON) != string(expectedJSON) {
		t.Fatalf("JSON mismatch\nactual:   %s\nexpected: %s", actualJSON, expectedJSON)
	}
}
