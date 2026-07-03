package lilycloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/crevas/Apple-Ads-CLI/internal/config"
)

const ProductName = "Lily Ads Revenue Analytics"

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type RevenueQuery struct {
	AppID               string `json:"appId"`
	From                string `json:"from"`
	To                  string `json:"to"`
	AppleAdsProvider    string `json:"appleAdsProvider,omitempty"`
	AppleAdsOrgID       string `json:"appleAdsOrgId,omitempty"`
	AppleAdsAdAccountID string `json:"appleAdsAdAccountId,omitempty"`
}

type RevenueStatus struct {
	Source              string   `json:"source"`
	Status              string   `json:"status"`
	Amount              *string  `json:"amount,omitempty"`
	Currency            *string  `json:"currency,omitempty"`
	ROAS                *float64 `json:"roas,omitempty"`
	MissingCapabilities []string `json:"missingCapabilities,omitempty"`
	Notice              string   `json:"notice,omitempty"`
	Raw                 any      `json:"raw,omitempty"`
}

type AuthStatus struct {
	Source                        string   `json:"source"`
	LoggedIn                      bool     `json:"loggedIn"`
	Configured                    bool     `json:"configured"`
	BaseURL                       string   `json:"baseUrl"`
	TokenHint                     string   `json:"tokenHint,omitempty"`
	Scope                         string   `json:"scope"`
	RequiredForAppleAdsOperations bool     `json:"requiredForAppleAdsOperations"`
	OptionalFor                   []string `json:"optionalFor"`
	NextSteps                     []string `json:"nextSteps"`
}

func New(cfg config.Config) Client {
	return Client{
		BaseURL:    strings.TrimRight(cfg.LilyCloudBase, "/"),
		Token:      strings.TrimSpace(cfg.LilyToken),
		HTTPClient: &http.Client{Timeout: cfg.Timeout()},
	}
}

func (c Client) AuthStatus() AuthStatus {
	return AuthStatus{
		Source:                        ProductName,
		LoggedIn:                      c.Token != "",
		Configured:                    c.Token != "",
		BaseURL:                       c.BaseURL,
		TokenHint:                     tokenHint(c.Token),
		Scope:                         "optional_revenue_analytics",
		RequiredForAppleAdsOperations: false,
		OptionalFor: []string{
			"keyword-level revenue analytics",
			"paid-user and ROAS enrichment",
			"Lily Ads Revenue Analytics cloud reports",
		},
		NextSteps: []string{
			"To manage Apple Ads, configure Apple Ads API credentials locally and run `lily ads doctor`. Private keys stay on this machine.",
			"Optional: run `lily login --token <token>` only when you want Lily Ads Revenue Analytics revenue and ROAS enrichment.",
		},
	}
}

func (c Client) RevenueSummary(query RevenueQuery) RevenueStatus {
	if c.Token == "" {
		return LoginRequired()
	}

	body, err := json.Marshal(query)
	if err != nil {
		return integrationError(fmt.Sprintf("Failed to encode revenue query: %v", err))
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/ads/revenue/query", bytes.NewReader(body))
	if err != nil {
		return integrationError(fmt.Sprintf("Failed to create revenue request: %v", err))
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return integrationError(fmt.Sprintf("Could not reach Lily Ads Revenue Analytics: %v", err))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return integrationError(fmt.Sprintf("Failed to read Lily response: %v", err))
	}
	var parsed any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return RevenueStatus{
			Source: ProductName,
			Status: "ok",
			Raw:    parsed,
		}
	case http.StatusUnauthorized:
		return LoginRequired()
	case http.StatusPaymentRequired, http.StatusForbidden:
		return RevenueStatus{
			Source: ProductName,
			Status: "dashboard_required",
			MissingCapabilities: []string{
				"lily_ads_revenue_analytics",
				"revenue_read",
			},
			Notice: "Revenue data requires Lily login and Lily Ads Revenue Analytics activation, so paid-user status and ROAS cannot be calculated.",
			Raw:    parsed,
		}
	case http.StatusConflict:
		return RevenueStatus{
			Source: ProductName,
			Status: "account_mismatch",
			MissingCapabilities: []string{
				"matching_lily_revenue_account",
			},
			Notice: "Apple Ads data was read from local credentials, but this Lily token is not authorized for the same Apple Ads account/app. Revenue and ROAS were skipped.",
			Raw:    parsed,
		}
	default:
		return RevenueStatus{
			Source: ProductName,
			Status: "unavailable",
			Notice: fmt.Sprintf("Lily Ads Revenue Analytics returned HTTP %d, so ROAS enrichment was skipped.", resp.StatusCode),
			Raw:    parsed,
		}
	}
}

func LoginRequired() RevenueStatus {
	return RevenueStatus{
		Source: ProductName,
		Status: "login_required",
		MissingCapabilities: []string{
			"lily_login",
			"lily_ads_revenue_analytics",
			"revenue_read",
		},
		Notice: "Revenue data requires Lily login and Lily Ads Revenue Analytics activation, so paid-user status and ROAS cannot be calculated.",
	}
}

func integrationError(message string) RevenueStatus {
	return RevenueStatus{
		Source: ProductName,
		Status: "integration_error",
		Notice: message,
	}
}

func tokenHint(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return token[:1] + "..."
	}
	return token[:6] + "..." + token[len(token)-4:]
}
