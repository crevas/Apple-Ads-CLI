package appleads

import (
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type PlanCreateInput struct {
	Name                     string
	AppID                    string
	Countries                []string
	Currency                 string
	DailyBudget              string
	AdGroupName              string
	DefaultBid               string
	CPAGoal                  string
	Keywords                 []KeywordPlan
	CampaignNegativeKeywords []NegativeKeywordPlan
	AdGroupNegativeKeywords  []NegativeKeywordPlan
	Creative                 CreativeSelection
	StartTime                string
	EndTime                  string
	Status                   string
	Supply                   string
	AllowPartial             bool
	Execute                  bool
	ProviderName             string
	CorrelationID            string
}

type KeywordPlan struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
	Bid       Money  `json:"bid"`
}

type NegativeKeywordPlan struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
}

type CreativeSelection struct {
	Kind          string `json:"kind,omitempty"`
	CreativeID    string `json:"creativeId,omitempty"`
	ProductPageID string `json:"productPageId,omitempty"`
	Name          string `json:"name,omitempty"`
	AdName        string `json:"adName,omitempty"`
}

type PlannedRequest struct {
	Step   string `json:"step"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   any    `json:"body,omitempty"`
}

type ExecutedStep struct {
	Step     string `json:"step"`
	ID       string `json:"id,omitempty"`
	Response any    `json:"response,omitempty"`
}

type PlanCreateResult struct {
	Tool           string           `json:"tool"`
	Provider       string           `json:"provider"`
	Mode           string           `json:"mode"`
	CorrelationID  string           `json:"correlationId,omitempty"`
	Planned        []PlannedRequest `json:"planned"`
	Executed       []ExecutedStep   `json:"executed,omitempty"`
	NextActions    []string         `json:"nextActions,omitempty"`
	SafetyReminder string           `json:"safetyReminder,omitempty"`
}

type CampaignCreate struct {
	Name        string
	AppID       string
	Countries   []string
	Currency    string
	DailyBudget string
	StartTime   string
	EndTime     string
	Status      string
	Supply      string
}

type AdGroupCreate struct {
	CampaignID string
	Name       string
	Currency   string
	Bid        string
	CPAGoal    string
	StartTime  string
	EndTime    string
	Status     string
}

type KeywordCreate struct {
	CampaignID string
	AdGroupID  string
	Text       string
	MatchType  string
	Currency   string
	Bid        string
	Status     string
}

type NegativeKeywordCreate struct {
	CampaignID string
	AdGroupID  string
	Text       string
	MatchType  string
	Status     string
}

type CreativeCreate struct {
	AppID         string
	ProductPageID string
	Name          string
}

type AdCreate struct {
	CampaignID string
	AdGroupID  string
	CreativeID string
	Name       string
	Status     string
}

type CampaignReportQuery struct {
	AppID       string
	From        string
	To          string
	TimeZone    string
	Granularity string
	Limit       int
	Offset      int
}

type Provider interface {
	Name() string
	PlannedRequests(input PlanCreateInput) []PlannedRequest
	CreateCampaign(ctx RequestContext, input CampaignCreate) (RawResponse, string, error)
	CreateAdGroup(ctx RequestContext, input AdGroupCreate) (RawResponse, string, error)
	BulkCreateKeywords(ctx RequestContext, keywords []KeywordCreate, allowPartial bool) (RawResponse, error)
	BulkCreateNegativeKeywords(ctx RequestContext, keywords []NegativeKeywordCreate, allowPartial bool) (RawResponse, error)
	CreateCreative(ctx RequestContext, input CreativeCreate) (RawResponse, string, error)
	CreateAd(ctx RequestContext, input AdCreate) (RawResponse, string, error)
	QueryCampaignReport(ctx RequestContext, input CampaignReportQuery) (RawResponse, error)
}

type RequestContext interface {
	Do(method string, path string, body any) (RawResponse, error)
}

type RawResponse map[string]any

func NormalizePlan(input PlanCreateInput) (PlanCreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.AppID = strings.TrimSpace(input.AppID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.DailyBudget = strings.TrimSpace(input.DailyBudget)
	input.AdGroupName = strings.TrimSpace(input.AdGroupName)
	input.DefaultBid = strings.TrimSpace(input.DefaultBid)
	input.CPAGoal = strings.TrimSpace(input.CPAGoal)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.Supply = strings.ToUpper(strings.TrimSpace(input.Supply))
	input.Creative.Kind = strings.ToLower(strings.TrimSpace(input.Creative.Kind))
	input.Creative.CreativeID = strings.TrimSpace(input.Creative.CreativeID)
	input.Creative.ProductPageID = strings.TrimSpace(input.Creative.ProductPageID)
	input.Creative.Name = strings.TrimSpace(input.Creative.Name)
	input.Creative.AdName = strings.TrimSpace(input.Creative.AdName)

	if input.Name == "" {
		return input, fmt.Errorf("--name is required")
	}
	if input.AppID == "" {
		return input, fmt.Errorf("--app-id is required")
	}
	if _, err := strconv.ParseInt(input.AppID, 10, 64); err != nil {
		return input, fmt.Errorf("--app-id must be a numeric App Store adamId")
	}
	if len(input.Countries) == 0 {
		return input, fmt.Errorf("--country or --countries is required")
	}
	input.Countries = NormalizeCountries(input.Countries)
	if input.Currency == "" {
		input.Currency = "USD"
	}
	if input.DailyBudget == "" {
		return input, fmt.Errorf("--daily-budget is required")
	}
	if input.AdGroupName == "" {
		input.AdGroupName = input.Name + " - Search Results"
	}
	if input.DefaultBid == "" {
		return input, fmt.Errorf("--bid is required")
	}
	if len(input.Keywords) == 0 {
		return input, fmt.Errorf("--keywords, --exact-keywords, or --broad-keywords is required")
	}
	if input.Status == "" {
		input.Status = "ENABLED"
	}
	if input.Supply == "" {
		input.Supply = "APPSTORE_SEARCH_RESULTS"
	}
	if input.Creative.Kind == "" && input.Creative.CreativeID != "" {
		input.Creative.Kind = "creative-id"
	}
	if input.Creative.Kind == "" && input.Creative.ProductPageID != "" {
		input.Creative.Kind = "product-page"
	}
	switch input.Creative.Kind {
	case "", "none", "default", "product-page", "creative-id":
	default:
		return input, fmt.Errorf("invalid --creative value %q", input.Creative.Kind)
	}

	for index := range input.Keywords {
		input.Keywords[index].Text = strings.TrimSpace(input.Keywords[index].Text)
		input.Keywords[index].MatchType = strings.ToUpper(strings.TrimSpace(input.Keywords[index].MatchType))
		input.Keywords[index].Bid.Currency = input.Currency
		if input.Keywords[index].Bid.Amount == "" {
			input.Keywords[index].Bid.Amount = input.DefaultBid
		}
		if input.Keywords[index].MatchType == "" {
			input.Keywords[index].MatchType = "EXACT"
		}
		if input.Keywords[index].Text == "" {
			return input, fmt.Errorf("keyword %d is empty", index+1)
		}
		if input.Keywords[index].MatchType != "EXACT" && input.Keywords[index].MatchType != "BROAD" {
			return input, fmt.Errorf("keyword %q has invalid match type %q", input.Keywords[index].Text, input.Keywords[index].MatchType)
		}
	}
	for index := range input.CampaignNegativeKeywords {
		if err := normalizeNegative(&input.CampaignNegativeKeywords[index], index); err != nil {
			return input, err
		}
	}
	for index := range input.AdGroupNegativeKeywords {
		if err := normalizeNegative(&input.AdGroupNegativeKeywords[index], index); err != nil {
			return input, err
		}
	}

	return input, nil
}

func NormalizeCountries(values []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range values {
		for _, item := range strings.Split(raw, ",") {
			code := strings.ToUpper(strings.TrimSpace(item))
			switch code {
			case "":
				continue
			case "UK":
				code = "GB"
			}
			if !seen[code] {
				seen[code] = true
				out = append(out, code)
			}
		}
	}
	return out
}

func ParseKeywords(raw string, matchType string, currency string, bid string) []KeywordPlan {
	var keywords []KeywordPlan
	for _, part := range strings.Split(raw, ",") {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		keywords = append(keywords, KeywordPlan{
			Text:      text,
			MatchType: strings.ToUpper(matchType),
			Bid:       Money{Amount: bid, Currency: strings.ToUpper(currency)},
		})
	}
	return keywords
}

func ParseNegativeKeywords(raw string, matchType string) []NegativeKeywordPlan {
	var keywords []NegativeKeywordPlan
	for _, part := range strings.Split(raw, ",") {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		keywords = append(keywords, NegativeKeywordPlan{
			Text:      text,
			MatchType: strings.ToUpper(matchType),
		})
	}
	return keywords
}

func normalizeNegative(keyword *NegativeKeywordPlan, index int) error {
	keyword.Text = strings.TrimSpace(keyword.Text)
	keyword.MatchType = strings.ToUpper(strings.TrimSpace(keyword.MatchType))
	if keyword.MatchType == "" {
		keyword.MatchType = "EXACT"
	}
	if keyword.Text == "" {
		return fmt.Errorf("negative keyword %d is empty", index+1)
	}
	if keyword.MatchType != "EXACT" && keyword.MatchType != "BROAD" {
		return fmt.Errorf("negative keyword %q has invalid match type %q", keyword.Text, keyword.MatchType)
	}
	return nil
}

func ExtractID(response RawResponse) string {
	for _, path := range [][]string{
		{"result", "id"},
		{"data", "id"},
		{"id"},
	} {
		if value, ok := nested(response, path...); ok {
			return stringifyID(value)
		}
	}
	return ""
}

func nested(value any, path ...string) (any, bool) {
	current := value
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func stringifyID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return fmt.Sprint(typed)
	}
}
