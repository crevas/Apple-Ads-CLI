package lilycloud

import "testing"

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
