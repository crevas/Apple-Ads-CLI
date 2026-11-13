package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultProviderAndExplicitOverrides(t *testing.T) {
	for _, tc := range []struct{ name, file, env, alias, want string }{
		{"new config", "", "", "", "platform"},
		{"old config without provider", "", "", "", "platform"},
		{"explicit v5 file", "campaignv5", "", "", "campaignv5"},
		{"environment overrides file", "campaignv5", "v1", "", "platform"},
		{"explicit v5 environment", "platform", "v5", "", "campaignv5"},
		{"legacy environment alias", "platform", "", "v5", "campaignv5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LILY_ADS_CONFIG", filepath.Join(t.TempDir(), "config.json"))
			t.Setenv("APPLE_ADS_PROVIDER", tc.env)
			t.Setenv("LILY_ADS_PROVIDER", tc.alias)
			if tc.name != "new config" {
				if err := Save(Config{Provider: tc.file, OrgID: "123"}); err != nil {
					t.Fatal(err)
				}
			}
			if got := Load().Provider; got != tc.want {
				t.Fatalf("provider = %q, want %q", got, tc.want)
			}
		})
	}
	if NormalizeProvider("") != "platform" || NormalizeProvider(" V1 ") != "platform" {
		t.Fatal("default and v1 alias must select platform")
	}
}

func TestV1NeverUsesLegacyOrgAsAdAccountID(t *testing.T) {
	cfg := Config{OrgID: "123"}
	if err := cfg.ValidateProviderScope(); err == nil || !strings.Contains(err.Error(), "APPLE_ADS_AD_ACCOUNT_ID") || !strings.Contains(err.Error(), "--provider campaignv5") {
		t.Fatalf("missing v1 account must explain migration: %v", err)
	}
	cfg.Provider = "v5"
	if err := cfg.ValidateProviderScope(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveLilyTokenDoesNotPersistAppleAdsEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apple-ads.json")
	t.Setenv("LILY_ADS_CONFIG", path)
	t.Setenv("APPLE_ADS_CLIENT_ID", "env-client")
	t.Setenv("APPLE_ADS_TEAM_ID", "env-team")
	t.Setenv("APPLE_ADS_KEY_ID", "env-key")
	t.Setenv("APPLE_ADS_ORG_ID", "env-org")
	t.Setenv("APPLE_ADS_PRIVATE_KEY_PATH", "/env/AuthKey.p8")

	if err := SaveLilyToken("lily-token"); err != nil {
		t.Fatalf("SaveLilyToken returned error: %v", err)
	}

	var stored Config
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if stored.LilyToken != "lily-token" {
		t.Fatalf("lilyToken = %q, want lily-token", stored.LilyToken)
	}
	if stored.ClientID != "" || stored.TeamID != "" || stored.KeyID != "" || stored.OrgID != "" || stored.PrivateKeyPath != "" {
		t.Fatalf("Apple Ads env credentials were persisted: %#v", stored)
	}
}

func TestClearLilyTokenPreservesFileAppleAdsConfigWithoutPersistingEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apple-ads.json")
	t.Setenv("LILY_ADS_CONFIG", path)
	t.Setenv("APPLE_ADS_CLIENT_ID", "env-client")

	initial := Config{
		LilyToken:      "lily-token",
		ClientID:       "file-client",
		TeamID:         "file-team",
		KeyID:          "file-key",
		OrgID:          "file-org",
		PrivateKeyPath: "/file/AuthKey.p8",
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := ClearLilyToken(); err != nil {
		t.Fatalf("ClearLilyToken returned error: %v", err)
	}

	var stored Config
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if stored.LilyToken != "" {
		t.Fatalf("lilyToken = %q, want empty", stored.LilyToken)
	}
	if stored.ClientID != "file-client" {
		t.Fatalf("clientId = %q, want file-client", stored.ClientID)
	}
}
