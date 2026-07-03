package lilycloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRevenueSummaryRequiresLogin(t *testing.T) {
	status := Client{}.RevenueSummary(RevenueQuery{
		AppID: "999999999",
		From:  "2026-06-01",
		To:    "2026-06-30",
	})

	if status.Source != ProductName {
		t.Fatalf("source = %q, want %q", status.Source, ProductName)
	}
	if status.Status != "login_required" {
		t.Fatalf("status = %q, want login_required", status.Status)
	}
	if status.Notice == "" {
		t.Fatal("notice is empty")
	}
	if len(status.MissingCapabilities) != 3 {
		t.Fatalf("missing capabilities = %v, want 3 entries", status.MissingCapabilities)
	}
}

func TestRevenueSummarySendsAppleAdsContext(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer lily-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	status := Client{
		BaseURL:    server.URL,
		Token:      "lily-token",
		HTTPClient: server.Client(),
	}.RevenueSummary(RevenueQuery{
		AppID:            "999999999",
		From:             "2026-06-01",
		To:               "2026-06-30",
		AppleAdsProvider: "campaignv5",
		AppleAdsOrgID:    "123456",
	})

	if status.Status != "ok" {
		t.Fatalf("status = %q, want ok", status.Status)
	}
	if body["appleAdsProvider"] != "campaignv5" {
		t.Fatalf("appleAdsProvider = %v, want campaignv5", body["appleAdsProvider"])
	}
	if body["appleAdsOrgId"] != "123456" {
		t.Fatalf("appleAdsOrgId = %v, want 123456", body["appleAdsOrgId"])
	}
}

func TestRevenueSummaryMapsAccountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"account_mismatch"}`))
	}))
	defer server.Close()

	status := Client{
		BaseURL:    server.URL,
		Token:      "lily-token",
		HTTPClient: server.Client(),
	}.RevenueSummary(RevenueQuery{
		AppID:         "999999999",
		AppleAdsOrgID: "other-org",
	})

	if status.Status != "account_mismatch" {
		t.Fatalf("status = %q, want account_mismatch", status.Status)
	}
	if status.Notice == "" {
		t.Fatal("notice is empty")
	}
}
