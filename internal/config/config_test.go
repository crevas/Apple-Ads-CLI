package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
