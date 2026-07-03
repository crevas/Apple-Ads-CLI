package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultProvider        = "campaignv5"
	DefaultCampaignV5Base  = "https://api.searchads.apple.com/api/v5"
	DefaultPlatformBase    = "https://api.ads.apple.com/v1"
	DefaultLilyCloudBase   = "https://www.chatlily.ai"
	DefaultAppleTokenURL   = "https://appleid.apple.com/auth/oauth2/token"
	DefaultConfigPathParts = ".config/lily/apple-ads.json"
)

type Config struct {
	Provider         string `json:"provider"`
	CampaignV5Base   string `json:"campaignV5Base"`
	PlatformBase     string `json:"platformBase"`
	LilyCloudBase    string `json:"lilyCloudBase"`
	LilyToken        string `json:"lilyToken"`
	TokenURL         string `json:"tokenUrl"`
	ClientID         string `json:"clientId"`
	TeamID           string `json:"teamId"`
	KeyID            string `json:"keyId"`
	OrgID            string `json:"orgId"`
	AdAccountID      string `json:"adAccountId"`
	PrivateKeyPath   string `json:"privateKeyPath"`
	DefaultCurrency  string `json:"defaultCurrency"`
	RequestTimeoutMs int    `json:"requestTimeoutMs"`
}

func Load() Config {
	cfg := Config{
		Provider:         DefaultProvider,
		CampaignV5Base:   DefaultCampaignV5Base,
		PlatformBase:     DefaultPlatformBase,
		LilyCloudBase:    DefaultLilyCloudBase,
		TokenURL:         DefaultAppleTokenURL,
		DefaultCurrency:  "USD",
		RequestTimeoutMs: 30000,
	}

	if fileCfg, err := loadFileConfig(); err == nil {
		cfg = merge(cfg, fileCfg)
	}

	cfg.Provider = firstNonEmpty(
		os.Getenv("APPLE_ADS_PROVIDER"),
		os.Getenv("LILY_ADS_PROVIDER"),
		cfg.Provider,
	)
	cfg.CampaignV5Base = firstNonEmpty(os.Getenv("APPLE_ADS_V5_BASE_URL"), os.Getenv("LILY_ADS_V5_BASE_URL"), cfg.CampaignV5Base)
	cfg.PlatformBase = firstNonEmpty(os.Getenv("APPLE_ADS_PLATFORM_BASE_URL"), os.Getenv("LILY_ADS_PLATFORM_BASE_URL"), cfg.PlatformBase)
	cfg.LilyCloudBase = firstNonEmpty(os.Getenv("LILY_CLOUD_BASE_URL"), cfg.LilyCloudBase)
	cfg.LilyToken = firstNonEmpty(os.Getenv("LILY_TOKEN"), os.Getenv("LILY_API_TOKEN"), cfg.LilyToken)
	cfg.TokenURL = firstNonEmpty(os.Getenv("APPLE_ADS_TOKEN_URL"), os.Getenv("LILY_ADS_TOKEN_URL"), cfg.TokenURL)
	cfg.ClientID = firstNonEmpty(os.Getenv("APPLE_ADS_CLIENT_ID"), os.Getenv("LILY_ADS_CLIENT_ID"), cfg.ClientID)
	cfg.TeamID = firstNonEmpty(os.Getenv("APPLE_ADS_TEAM_ID"), os.Getenv("LILY_ADS_TEAM_ID"), cfg.TeamID)
	cfg.KeyID = firstNonEmpty(os.Getenv("APPLE_ADS_KEY_ID"), os.Getenv("LILY_ADS_KEY_ID"), cfg.KeyID)
	cfg.OrgID = firstNonEmpty(os.Getenv("APPLE_ADS_ORG_ID"), os.Getenv("LILY_ADS_ORG_ID"), cfg.OrgID)
	cfg.AdAccountID = firstNonEmpty(os.Getenv("APPLE_ADS_AD_ACCOUNT_ID"), os.Getenv("LILY_ADS_AD_ACCOUNT_ID"), cfg.AdAccountID)
	cfg.PrivateKeyPath = firstNonEmpty(os.Getenv("APPLE_ADS_PRIVATE_KEY_PATH"), os.Getenv("LILY_ADS_PRIVATE_KEY_PATH"), cfg.PrivateKeyPath)
	cfg.DefaultCurrency = strings.ToUpper(firstNonEmpty(os.Getenv("APPLE_ADS_CURRENCY"), os.Getenv("LILY_ADS_CURRENCY"), cfg.DefaultCurrency))

	if raw := firstNonEmpty(os.Getenv("APPLE_ADS_TIMEOUT_MS"), os.Getenv("LILY_ADS_TIMEOUT_MS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			cfg.RequestTimeoutMs = parsed
		}
	}

	cfg.Provider = NormalizeProvider(cfg.Provider)
	return cfg
}

func Save(update Config) error {
	current := Load()
	merged := merge(current, update)
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func SaveLilyToken(token string) error {
	return Save(Config{LilyToken: strings.TrimSpace(token)})
}

func ClearLilyToken() error {
	current := Load()
	current.LilyToken = ""
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c Config) Timeout() time.Duration {
	if c.RequestTimeoutMs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.RequestTimeoutMs) * time.Millisecond
}

func (c Config) ValidateAuth() error {
	var missing []string
	for name, value := range map[string]string{
		"APPLE_ADS_CLIENT_ID":        c.ClientID,
		"APPLE_ADS_TEAM_ID":          c.TeamID,
		"APPLE_ADS_KEY_ID":           c.KeyID,
		"APPLE_ADS_PRIVATE_KEY_PATH": c.PrivateKeyPath,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required Apple Ads local credential config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c Config) ValidateProviderScope() error {
	switch NormalizeProvider(c.Provider) {
	case "campaignv5":
		if strings.TrimSpace(c.OrgID) == "" {
			return errors.New("APPLE_ADS_ORG_ID is required for campaignv5 provider")
		}
	case "platform":
		if strings.TrimSpace(c.AdAccountID) == "" {
			return errors.New("APPLE_ADS_AD_ACCOUNT_ID is required for platform provider")
		}
	default:
		return fmt.Errorf("unsupported provider %q", c.Provider)
	}
	return nil
}

func ConfigPath() string {
	if raw := os.Getenv("LILY_ADS_CONFIG"); raw != "" {
		return raw
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfigPathParts
	}
	return filepath.Join(home, DefaultConfigPathParts)
}

func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "v5", "legacy", "campaign-management-v5", "campaignv5":
		return "campaignv5"
	case "platform", "platform-preview", "platformv1", "v1":
		return "platform"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func loadFileConfig() (Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func merge(base Config, override Config) Config {
	if override.Provider != "" {
		base.Provider = override.Provider
	}
	if override.CampaignV5Base != "" {
		base.CampaignV5Base = override.CampaignV5Base
	}
	if override.PlatformBase != "" {
		base.PlatformBase = override.PlatformBase
	}
	if override.LilyCloudBase != "" {
		base.LilyCloudBase = override.LilyCloudBase
	}
	if override.LilyToken != "" {
		base.LilyToken = override.LilyToken
	}
	if override.TokenURL != "" {
		base.TokenURL = override.TokenURL
	}
	if override.ClientID != "" {
		base.ClientID = override.ClientID
	}
	if override.TeamID != "" {
		base.TeamID = override.TeamID
	}
	if override.KeyID != "" {
		base.KeyID = override.KeyID
	}
	if override.OrgID != "" {
		base.OrgID = override.OrgID
	}
	if override.AdAccountID != "" {
		base.AdAccountID = override.AdAccountID
	}
	if override.PrivateKeyPath != "" {
		base.PrivateKeyPath = override.PrivateKeyPath
	}
	if override.DefaultCurrency != "" {
		base.DefaultCurrency = override.DefaultCurrency
	}
	if override.RequestTimeoutMs > 0 {
		base.RequestTimeoutMs = override.RequestTimeoutMs
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
