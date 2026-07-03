package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
	"github.com/crevas/Apple-Ads-CLI/internal/config"
	"github.com/crevas/Apple-Ads-CLI/internal/lilycloud"
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

func TestAuthStatusExplainsLilyLoginIsOptional(t *testing.T) {
	t.Setenv("LILY_ADS_CONFIG", t.TempDir()+"/apple-ads.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"auth", "status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned code %d, stderr: %s", code, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("auth status output is not JSON: %v\n%s", err, stdout.String())
	}

	if got := payload["requiredForAppleAdsOperations"]; got != false {
		t.Fatalf("requiredForAppleAdsOperations = %v, want false", got)
	}
	steps, ok := payload["nextSteps"].([]any)
	if !ok {
		t.Fatalf("nextSteps missing or invalid: %#v", payload["nextSteps"])
	}
	joinedSteps := joinAnyStrings(steps)
	if !strings.Contains(joinedSteps, "lily ads doctor") {
		t.Fatalf("nextSteps = %q, want lily ads doctor guidance", joinedSteps)
	}
	if !strings.Contains(joinedSteps, "Private keys stay on this machine") {
		t.Fatalf("nextSteps = %q, want local private-key guidance", joinedSteps)
	}
}

func TestDoctorSeparatesAppleAdsCredentialsFromLilyLogin(t *testing.T) {
	t.Setenv("LILY_ADS_CONFIG", t.TempDir()+"/apple-ads.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"ads", "doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned code %d, stderr: %s", code, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("doctor output is not JSON: %v\n%s", err, stdout.String())
	}
	if _, ok := payload["auth"]; ok {
		t.Fatalf("doctor output should not expose ambiguous top-level auth field: %#v", payload["auth"])
	}

	credentials, ok := payload["appleAdsCredentials"].(map[string]any)
	if !ok {
		t.Fatalf("appleAdsCredentials missing or invalid: %#v", payload["appleAdsCredentials"])
	}
	if got := credentials["configured"]; got != false {
		t.Fatalf("appleAdsCredentials.configured = %v, want false", got)
	}
	if got := credentials["privateKeyUploaded"]; got != false {
		t.Fatalf("appleAdsCredentials.privateKeyUploaded = %v, want false", got)
	}
	if got := credentials["error"]; !strings.Contains(toString(got), "Apple Ads local credential") {
		t.Fatalf("appleAdsCredentials.error = %v, want local credential wording", got)
	}
	if got := credentials["error"]; !strings.Contains(toString(got), "APPLE_ADS_CLIENT_ID") {
		t.Fatalf("appleAdsCredentials.error = %v, want APPLE_ADS_CLIENT_ID", got)
	}
	if got := credentials["error"]; strings.Contains(toString(got), "LILY_ADS_CLIENT_ID") {
		t.Fatalf("appleAdsCredentials.error = %v, should not use Lily prefix for Apple credentials", got)
	}

	lilyLogin, ok := payload["lilyLogin"].(map[string]any)
	if !ok {
		t.Fatalf("lilyLogin missing or invalid: %#v", payload["lilyLogin"])
	}
	if got := lilyLogin["requiredForAppleAdsOperations"]; got != false {
		t.Fatalf("lilyLogin.requiredForAppleAdsOperations = %v, want false", got)
	}
}

func TestRevenueQueryWithAppleAdsContextUsesProviderScope(t *testing.T) {
	v5 := revenueQueryWithAppleAdsContext(config.Config{
		Provider:    "campaignv5",
		OrgID:       "123456",
		AdAccountID: "ad-account",
	}, lilycloud.RevenueQuery{AppID: "999999999"})
	if v5.AppleAdsProvider != "campaignv5" {
		t.Fatalf("v5 provider = %q, want campaignv5", v5.AppleAdsProvider)
	}
	if v5.AppleAdsOrgID != "123456" {
		t.Fatalf("v5 org id = %q, want 123456", v5.AppleAdsOrgID)
	}
	if v5.AppleAdsAdAccountID != "" {
		t.Fatalf("v5 ad account id = %q, want empty", v5.AppleAdsAdAccountID)
	}

	platform := revenueQueryWithAppleAdsContext(config.Config{
		Provider:    "platform",
		OrgID:       "123456",
		AdAccountID: "ad-account",
	}, lilycloud.RevenueQuery{AppID: "999999999"})
	if platform.AppleAdsProvider != "platform" {
		t.Fatalf("platform provider = %q, want platform", platform.AppleAdsProvider)
	}
	if platform.AppleAdsAdAccountID != "ad-account" {
		t.Fatalf("platform ad account id = %q, want ad-account", platform.AppleAdsAdAccountID)
	}
	if platform.AppleAdsOrgID != "" {
		t.Fatalf("platform org id = %q, want empty", platform.AppleAdsOrgID)
	}
}

func joinAnyStrings(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, toString(value))
	}
	return strings.Join(parts, "\n")
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
