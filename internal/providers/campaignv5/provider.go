package campaignv5

import (
	"fmt"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

type Provider struct {
	OrgID string
}

func New(orgID string) Provider {
	return Provider{OrgID: orgID}
}

func (Provider) Name() string {
	return "campaignv5"
}

func (p Provider) PlannedRequests(input appleads.PlanCreateInput) []appleads.PlannedRequest {
	campaign := p.campaignPayload(appleads.CampaignCreate{
		Name:        input.Name,
		AppID:       input.AppID,
		Countries:   input.Countries,
		Currency:    input.Currency,
		DailyBudget: input.DailyBudget,
		Status:      input.Status,
		Supply:      input.Supply,
	})
	adGroup := adGroupPayload(appleads.AdGroupCreate{
		CampaignID: "$campaign.id",
		Name:       input.AdGroupName,
		Currency:   input.Currency,
		Bid:        input.DefaultBid,
		CPAGoal:    input.CPAGoal,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Status:     input.Status,
	})
	var keywords []map[string]any
	for _, keyword := range input.Keywords {
		keywords = append(keywords, keywordPayload(appleads.KeywordCreate{
			Text:      keyword.Text,
			MatchType: keyword.MatchType,
			Currency:  input.Currency,
			Bid:       keyword.Bid.Amount,
			Status:    "ACTIVE",
		}))
	}
	planned := []appleads.PlannedRequest{
		{Step: "create_campaign", Method: "POST", Path: "/campaigns", Body: campaign},
		{Step: "create_ad_group", Method: "POST", Path: "/campaigns/$campaign.id/adgroups", Body: adGroup},
		{Step: "bulk_create_keywords", Method: "POST", Path: "/campaigns/$campaign.id/adgroups/$adGroup.id/targetingkeywords/bulk", Body: keywords},
	}
	if len(input.CampaignNegativeKeywords) > 0 {
		planned = append(planned, appleads.PlannedRequest{
			Step: "bulk_create_campaign_negative_keywords", Method: "POST", Path: "/campaigns/$campaign.id/negativekeywords/bulk",
			Body: negativePayloads(input.CampaignNegativeKeywords),
		})
	}
	if len(input.AdGroupNegativeKeywords) > 0 {
		planned = append(planned, appleads.PlannedRequest{
			Step: "bulk_create_adgroup_negative_keywords", Method: "POST", Path: "/campaigns/$campaign.id/adgroups/$adGroup.id/negativekeywords/bulk",
			Body: negativePayloads(input.AdGroupNegativeKeywords),
		})
	}
	if input.Creative.Kind != "" && input.Creative.Kind != "none" {
		creativeID := input.Creative.CreativeID
		if input.Creative.Kind == "default" || input.Creative.Kind == "product-page" {
			creativeID = "$creative.id"
			planned = append(planned, appleads.PlannedRequest{
				Step: "create_creative", Method: "POST", Path: "/creatives",
				Body: creativePayload(appleads.CreativeCreate{
					AppID:         input.AppID,
					ProductPageID: input.Creative.ProductPageID,
					Name:          defaultCreativeName(input),
				}),
			})
		}
		planned = append(planned, appleads.PlannedRequest{
			Step: "create_ad", Method: "POST", Path: "/campaigns/$campaign.id/adgroups/$adGroup.id/ads",
			Body: adPayload(appleads.AdCreate{
				CreativeID: creativeID,
				Name:       defaultAdName(input),
				Status:     input.Status,
			}),
		})
	}
	return planned
}

func (p Provider) CreateCampaign(ctx appleads.RequestContext, input appleads.CampaignCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/campaigns", p.campaignPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) CreateAdGroup(ctx appleads.RequestContext, input appleads.AdGroupCreate) (appleads.RawResponse, string, error) {
	path := fmt.Sprintf("/campaigns/%s/adgroups", input.CampaignID)
	resp, err := ctx.Do("POST", path, adGroupPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) BulkCreateKeywords(ctx appleads.RequestContext, keywords []appleads.KeywordCreate, _ bool) (appleads.RawResponse, error) {
	if len(keywords) == 0 {
		return appleads.RawResponse{}, nil
	}
	campaignID := keywords[0].CampaignID
	adGroupID := keywords[0].AdGroupID
	if campaignID == "" || adGroupID == "" {
		return nil, fmt.Errorf("campaign id and ad group id are required")
	}
	var body []map[string]any
	for _, keyword := range keywords {
		body = append(body, keywordPayload(keyword))
	}
	return ctx.Do("POST", fmt.Sprintf("/campaigns/%s/adgroups/%s/targetingkeywords/bulk", campaignID, adGroupID), body)
}

func (Provider) BulkCreateNegativeKeywords(ctx appleads.RequestContext, keywords []appleads.NegativeKeywordCreate, _ bool) (appleads.RawResponse, error) {
	if len(keywords) == 0 {
		return appleads.RawResponse{}, nil
	}
	campaignID := keywords[0].CampaignID
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	var body []map[string]any
	for _, keyword := range keywords {
		body = append(body, negativePayload(keyword.Text, keyword.MatchType))
	}
	if keywords[0].AdGroupID != "" {
		return ctx.Do("POST", fmt.Sprintf("/campaigns/%s/adgroups/%s/negativekeywords/bulk", campaignID, keywords[0].AdGroupID), body)
	}
	return ctx.Do("POST", fmt.Sprintf("/campaigns/%s/negativekeywords/bulk", campaignID), body)
}

func (Provider) CreateCreative(ctx appleads.RequestContext, input appleads.CreativeCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/creatives", creativePayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) CreateAd(ctx appleads.RequestContext, input appleads.AdCreate) (appleads.RawResponse, string, error) {
	path := fmt.Sprintf("/campaigns/%s/adgroups/%s/ads", input.CampaignID, input.AdGroupID)
	resp, err := ctx.Do("POST", path, adPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) QueryCampaignReport(ctx appleads.RequestContext, input appleads.CampaignReportQuery) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/reports/campaigns", campaignReportPayload(input))
}

func (p Provider) campaignPayload(input appleads.CampaignCreate) map[string]any {
	return map[string]any{
		"name":               input.Name,
		"orgId":              parseIDOrString(p.OrgID),
		"adamId":             parseIDOrString(input.AppID),
		"countriesOrRegions": input.Countries,
		"dailyBudgetAmount":  appleads.Money{Amount: input.DailyBudget, Currency: input.Currency},
		"supplySources":      []string{defaultString(input.Supply, "APPSTORE_SEARCH_RESULTS")},
		"adChannelType":      "SEARCH",
		"billingEvent":       "TAPS",
		"status":             defaultString(input.Status, "ENABLED"),
	}
}

func adGroupPayload(input appleads.AdGroupCreate) map[string]any {
	payload := map[string]any{
		"name":                   input.Name,
		"defaultBidAmount":       appleads.Money{Amount: input.Bid, Currency: input.Currency},
		"pricingModel":           "CPC",
		"automatedKeywordsOptIn": false,
		"status":                 defaultString(input.Status, "ENABLED"),
	}
	if input.StartTime != "" {
		payload["startTime"] = input.StartTime
	}
	if input.EndTime != "" {
		payload["endTime"] = input.EndTime
	}
	if input.CPAGoal != "" {
		payload["cpaGoal"] = appleads.Money{Amount: input.CPAGoal, Currency: input.Currency}
	}
	return payload
}

func keywordPayload(input appleads.KeywordCreate) map[string]any {
	return map[string]any{
		"text":      input.Text,
		"matchType": input.MatchType,
		"status":    legacyKeywordStatus(input.Status),
		"bidAmount": appleads.Money{Amount: input.Bid, Currency: input.Currency},
	}
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func parseIDOrString(value string) any {
	var parsed int64
	if _, err := fmt.Sscan(value, &parsed); err == nil {
		return parsed
	}
	return value
}

func legacyKeywordStatus(status string) string {
	switch status {
	case "", "ENABLED":
		return "ACTIVE"
	default:
		return status
	}
}

func negativePayloads(keywords []appleads.NegativeKeywordPlan) []map[string]any {
	payloads := make([]map[string]any, 0, len(keywords))
	for _, keyword := range keywords {
		payloads = append(payloads, negativePayload(keyword.Text, keyword.MatchType))
	}
	return payloads
}

func negativePayload(text string, matchType string) map[string]any {
	return map[string]any{
		"text":      text,
		"matchType": matchType,
	}
}

func creativePayload(input appleads.CreativeCreate) map[string]any {
	creativeType := "DEFAULT_PRODUCT_PAGE"
	destination := map[string]any{"adamId": parseIDOrString(input.AppID)}
	if input.ProductPageID != "" {
		creativeType = "CUSTOM_PRODUCT_PAGE"
		destination["productPageId"] = input.ProductPageID
	}
	return map[string]any{
		"name":         input.Name,
		"creativeType": creativeType,
		"destination":  destination,
	}
}

func adPayload(input appleads.AdCreate) map[string]any {
	return map[string]any{
		"name":       input.Name,
		"status":     defaultString(input.Status, "ENABLED"),
		"creativeId": parseIDOrString(input.CreativeID),
	}
}

func campaignReportPayload(input appleads.CampaignReportQuery) map[string]any {
	conditions := []map[string]any{}
	if input.AppID != "" {
		conditions = append(conditions, map[string]any{
			"field":    "adamId",
			"operator": "EQUALS",
			"values":   []string{input.AppID},
		})
	}
	return map[string]any{
		"startTime":                  input.From,
		"endTime":                    input.To,
		"granularity":                defaultString(input.Granularity, "DAILY"),
		"timeZone":                   defaultString(input.TimeZone, "ORTZ"),
		"returnRecordsWithNoMetrics": false,
		"returnRowTotals":            false,
		"returnGrandTotals":          true,
		"selector": map[string]any{
			"orderBy": []map[string]any{
				{"field": "localSpend", "sortOrder": "DESCENDING"},
			},
			"pagination": map[string]any{
				"offset": input.Offset,
				"limit":  defaultInt(input.Limit, 1000),
			},
			"conditions": conditions,
		},
	}
}

func defaultInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func defaultCreativeName(input appleads.PlanCreateInput) string {
	if input.Creative.Name != "" {
		return input.Creative.Name
	}
	return input.Name + " Creative"
}

func defaultAdName(input appleads.PlanCreateInput) string {
	if input.Creative.AdName != "" {
		return input.Creative.AdName
	}
	return input.Name + " Ad"
}
