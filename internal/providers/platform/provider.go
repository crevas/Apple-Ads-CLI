package platform

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

type Provider struct{}

func New() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "platform"
}

func (p Provider) PlannedRequests(input appleads.PlanCreateInput) []appleads.PlannedRequest {
	campaign := campaignPayload(appleads.CampaignCreate{
		Name:        input.Name,
		AppID:       input.AppID,
		Countries:   input.Countries,
		Currency:    input.Currency,
		DailyBudget: input.DailyBudget,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
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
	var keywords []appleads.KeywordCreate
	for _, keyword := range input.Keywords {
		keywords = append(keywords, appleads.KeywordCreate{
			AdGroupID: "$adGroup.id",
			Text:      keyword.Text,
			MatchType: keyword.MatchType,
			Currency:  input.Currency,
			Bid:       keyword.Bid.Amount,
			Status:    input.Status,
		})
	}
	planned := []appleads.PlannedRequest{
		{Step: "create_campaign", Method: "POST", Path: "/campaigns", Body: campaign},
		{Step: "create_ad_group", Method: "POST", Path: "/adgroups", Body: adGroup},
		{Step: "bulk_create_keywords", Method: "POST", Path: "/keywords/bulk-create", Body: keywordBulkPayload(keywords, input.AllowPartial)},
	}
	if len(input.CampaignNegativeKeywords) > 0 {
		planned = append(planned, appleads.PlannedRequest{
			Step: "bulk_create_campaign_negative_keywords", Method: "POST", Path: "/negative-keywords/bulk-create",
			Body: negativeBulkPayload(negativeCreates("$campaign.id", "", input.CampaignNegativeKeywords, input.Status), input.AllowPartial),
		})
	}
	if len(input.AdGroupNegativeKeywords) > 0 {
		planned = append(planned, appleads.PlannedRequest{
			Step: "bulk_create_adgroup_negative_keywords", Method: "POST", Path: "/negative-keywords/bulk-create",
			Body: negativeBulkPayload(negativeCreates("$campaign.id", "$adGroup.id", input.AdGroupNegativeKeywords, input.Status), input.AllowPartial),
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
			Step: "create_ad", Method: "POST", Path: "/ads",
			Body: adPayload(appleads.AdCreate{
				AdGroupID:  "$adGroup.id",
				CreativeID: creativeID,
				Name:       defaultAdName(input),
				Status:     input.Status,
			}),
		})
	}
	return planned
}

func (Provider) CreateCampaign(ctx appleads.RequestContext, input appleads.CampaignCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/campaigns", campaignPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) CreateAdGroup(ctx appleads.RequestContext, input appleads.AdGroupCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/adgroups", adGroupPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) BulkCreateKeywords(ctx appleads.RequestContext, keywords []appleads.KeywordCreate, allowPartial bool) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/keywords/bulk-create", keywordBulkPayload(keywords, allowPartial))
}

func (Provider) BulkCreateNegativeKeywords(ctx appleads.RequestContext, keywords []appleads.NegativeKeywordCreate, allowPartial bool) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/negative-keywords/bulk-create", negativeBulkPayload(keywords, allowPartial))
}

func (Provider) CreateCreative(ctx appleads.RequestContext, input appleads.CreativeCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/creatives", creativePayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) CreateAd(ctx appleads.RequestContext, input appleads.AdCreate) (appleads.RawResponse, string, error) {
	resp, err := ctx.Do("POST", "/ads", adPayload(input))
	if err != nil {
		return nil, "", err
	}
	return resp, appleads.ExtractID(resp), nil
}

func (Provider) QueryCampaignReport(ctx appleads.RequestContext, input appleads.CampaignReportQuery) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/reports/apps/campaigns/query", campaignReportPayload(input))
}

func (Provider) QueryRecommendations(ctx appleads.RequestContext, input appleads.RecommendationQuery) (appleads.RawResponse, error) {
	path, err := recommendationPath(input.Type)
	if err != nil {
		return nil, err
	}
	return ctx.Do("POST", path, recommendationPayload(input))
}

func (Provider) QuerySuggestions(ctx appleads.RequestContext, input appleads.SuggestionQuery) (appleads.RawResponse, error) {
	path, err := suggestionPath(input.Type)
	if err != nil {
		return nil, err
	}
	return ctx.Do("POST", path, suggestionPayload(input))
}

func (Provider) QuerySearchTermPopularity(ctx appleads.RequestContext, input appleads.SearchTermPopularityQuery) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/insights/apps/search-term-popularity/query", searchTermPopularityPayload(input))
}

func (Provider) QueryImpressionShare(ctx appleads.RequestContext, input appleads.ImpressionShareQuery) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/insights/apps/impression-share/query", impressionSharePayload(input))
}

func (Provider) QueryChangeHistory(ctx appleads.RequestContext, input appleads.ChangeHistoryQuery) (appleads.RawResponse, error) {
	return ctx.Do("POST", "/change-history/query", changeHistoryPayload(input))
}

func (Provider) GetChangeHistoryDetail(ctx appleads.RequestContext, detailID string) (appleads.RawResponse, error) {
	return ctx.Do("GET", "/change-history/"+url.PathEscape(detailID), nil)
}

func campaignPayload(input appleads.CampaignCreate) map[string]any {
	payload := map[string]any{
		"name":               input.Name,
		"status":             defaultString(input.Status, "ENABLED"),
		"promotedObjectType": "APPSTORE_APP",
		"promotedObjectId":   input.AppID,
		"billingEvent":       "TAPS",
		"bidStrategy": map[string]any{
			"bidStrategyType": "MANUAL_CPT",
			"bidStrategyGoal": "TAPS",
		},
		"dailyBudget": map[string]any{
			"value": appleads.Money{Amount: input.DailyBudget, Currency: input.Currency},
		},
		"targeting": map[string]any{
			"supplyPlacement": map[string]any{"include": []string{defaultString(input.Supply, "APPSTORE_SEARCH_RESULTS")}},
			"countryOrRegion": map[string]any{"include": input.Countries},
		},
	}
	if input.StartTime != "" {
		payload["startTime"] = input.StartTime
	}
	if input.EndTime != "" {
		payload["endTime"] = input.EndTime
	}
	return payload
}

func adGroupPayload(input appleads.AdGroupCreate) map[string]any {
	payload := map[string]any{
		"campaignId":                parseIDOrToken(input.CampaignID),
		"name":                      input.Name,
		"status":                    defaultString(input.Status, "ENABLED"),
		"pricingModel":              "CPT",
		"automatedKeywordsOptIn":    false,
		"automatedKeywordsRequired": false,
		"bidStrategy": map[string]any{
			"bidStrategyType": "MANUAL_CPT",
			"bidStrategyGoal": "TAPS",
			"bid":             appleads.Money{Amount: input.Bid, Currency: input.Currency},
		},
	}
	if input.CPAGoal != "" {
		payload["targetCPA"] = appleads.Money{Amount: input.CPAGoal, Currency: input.Currency}
	}
	if input.StartTime != "" {
		payload["startTime"] = input.StartTime
	}
	if input.EndTime != "" {
		payload["endTime"] = input.EndTime
	}
	return payload
}

func keywordBulkPayload(keywords []appleads.KeywordCreate, allowPartial bool) map[string]any {
	operations := make([]map[string]any, 0, len(keywords))
	for _, keyword := range keywords {
		operations = append(operations, map[string]any{
			"adGroupId": parseIDOrToken(keyword.AdGroupID),
			"text":      keyword.Text,
			"status":    defaultString(keyword.Status, "ENABLED"),
			"matchType": keyword.MatchType,
			"bid":       appleads.Money{Amount: keyword.Bid, Currency: keyword.Currency},
		})
	}
	return map[string]any{
		"allowPartialSuccess": allowPartial,
		"operations":          operations,
	}
}

func negativeCreates(campaignID string, adGroupID string, keywords []appleads.NegativeKeywordPlan, status string) []appleads.NegativeKeywordCreate {
	creates := make([]appleads.NegativeKeywordCreate, 0, len(keywords))
	for _, keyword := range keywords {
		creates = append(creates, appleads.NegativeKeywordCreate{
			CampaignID: campaignID,
			AdGroupID:  adGroupID,
			Text:       keyword.Text,
			MatchType:  keyword.MatchType,
			Status:     status,
		})
	}
	return creates
}

func negativeBulkPayload(keywords []appleads.NegativeKeywordCreate, allowPartial bool) map[string]any {
	operations := make([]map[string]any, 0, len(keywords))
	for _, keyword := range keywords {
		body := map[string]any{
			"text":      keyword.Text,
			"status":    defaultString(keyword.Status, "ENABLED"),
			"matchType": keyword.MatchType,
		}
		if keyword.AdGroupID != "" {
			body["adGroupId"] = parseIDOrToken(keyword.AdGroupID)
		} else {
			body["campaignId"] = parseIDOrToken(keyword.CampaignID)
		}
		operations = append(operations, body)
	}
	return map[string]any{
		"allowPartialSuccess": allowPartial,
		"operations":          operations,
	}
}

func creativePayload(input appleads.CreativeCreate) map[string]any {
	creativeType := "DEFAULT_PRODUCT_PAGE"
	parameters := map[string]any{"adamId": input.AppID}
	if input.ProductPageID != "" {
		creativeType = "CUSTOM_PRODUCT_PAGE"
		parameters["productPageId"] = input.ProductPageID
	}
	return map[string]any{
		"name":         input.Name,
		"creativeType": creativeType,
		"destination": map[string]any{
			"destinationType": "APP_STORE_PRODUCT_PAGE",
			"parameters":      parameters,
		},
	}
}

func adPayload(input appleads.AdCreate) map[string]any {
	return map[string]any{
		"adGroupId":  parseIDOrToken(input.AdGroupID),
		"name":       input.Name,
		"status":     defaultString(input.Status, "ENABLED"),
		"creativeId": parseIDOrToken(input.CreativeID),
	}
}

func campaignReportPayload(input appleads.CampaignReportQuery) map[string]any {
	filters := []map[string]any{}
	if input.AppID != "" {
		filters = append(filters, map[string]any{
			"field":    "promotedObjectId",
			"operator": "EQUALS",
			"value":    input.AppID,
		})
	}
	return map[string]any{
		"filters": filters,
		"sorting": []map[string]any{
			{"field": "localSpend", "order": "DESC"},
		},
		"pagination": map[string]any{
			"offset":   input.Offset,
			"pageSize": defaultInt(input.Limit, 100),
		},
		"timeRange": map[string]any{
			"start":       input.From,
			"end":         input.To,
			"timeZone":    defaultString(input.TimeZone, "ORTZ"),
			"granularity": defaultString(input.Granularity, "DAILY"),
		},
		"options": map[string]any{
			"includeRows": []string{"GRAND_TOTAL"},
		},
	}
}

func recommendationPath(recommendationType string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(recommendationType)) {
	case "KEYWORD":
		return "/recommendations/keywords/query", nil
	case "TARGET_CPA":
		return "/recommendations/target-cpas/query", nil
	case "DAILY_BUDGET":
		return "/recommendations/daily-budgets/query", nil
	default:
		return "", fmt.Errorf("unsupported Platform recommendation type %q", recommendationType)
	}
}

func suggestionPath(suggestionType string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(suggestionType)) {
	case "KEYWORD":
		return "/suggestions/keywords/query", nil
	case "TARGET_CPA":
		return "/suggestions/target-cpas/query", nil
	default:
		return "", fmt.Errorf("unsupported Platform suggestion type %q", suggestionType)
	}
}

func promotedObjectFilters(appID string, listValues bool) []map[string]any {
	appValue := any(appID)
	typeValue := any("APPSTORE_APP")
	if listValues {
		appValue = []string{appID}
		typeValue = []string{"APPSTORE_APP"}
	}
	return []map[string]any{
		{"field": "promotedObjectId", "operator": "EQUALS", "value": appValue},
		{"field": "promotedObjectType", "operator": "EQUALS", "value": typeValue},
	}
}

func recommendationPayload(input appleads.RecommendationQuery) map[string]any {
	filters := promotedObjectFilters(input.AppID, true)
	if input.State != "" {
		filters = append(filters, map[string]any{
			"field": "state", "operator": "EQUALS", "value": []string{input.State},
		})
	}
	payload := map[string]any{
		"filters": filters,
		"pagination": map[string]any{
			"offset": input.Offset, "pageSize": defaultInt(input.Limit, 50),
		},
	}
	return payload
}

func suggestionPayload(input appleads.SuggestionQuery) map[string]any {
	filters := promotedObjectFilters(input.AppID, true)
	if !strings.EqualFold(input.Type, "TARGET_CPA") {
		if len(input.Terms) > 0 {
			filters = append(filters, map[string]any{
				"field": "terms", "operator": "IN", "value": input.Terms,
			})
		}
		if len(input.Countries) > 0 {
			filters = append(filters, map[string]any{
				"field": "countriesOrRegions", "operator": "IN", "value": input.Countries,
			})
		}
	}
	payload := map[string]any{"filters": filters}
	if !strings.EqualFold(input.Type, "TARGET_CPA") {
		payload["pagination"] = map[string]any{
			"offset": input.Offset, "pageSize": defaultInt(input.Limit, 50),
		}
	}
	return payload
}

func searchTermPopularityPayload(input appleads.SearchTermPopularityQuery) map[string]any {
	return map[string]any{
		"filters": []map[string]any{
			{"field": "countryOrRegion", "operator": "EQUALS", "value": input.Country},
			{"field": "genre", "operator": "EQUALS", "value": normalizePopularityGenre(input.Genre)},
		},
		"timeRange": map[string]any{
			"start": input.From, "end": input.To, "granularity": input.Granularity,
		},
		"pagination": map[string]any{
			"offset": input.Offset, "pageSize": defaultInt(input.Limit, 100),
		},
	}
}

func impressionSharePayload(input appleads.ImpressionShareQuery) map[string]any {
	filters := []map[string]any{
		{"field": "promotedObjectId", "operator": "IN", "value": []string{input.AppID}},
	}
	if len(input.Countries) > 0 {
		filters = append(filters, map[string]any{
			"field": "countryOrRegion", "operator": "IN", "value": input.Countries,
		})
	}
	if len(input.SearchTerms) > 0 {
		filters = append(filters, map[string]any{
			"field": "searchTerm", "operator": "IN", "value": input.SearchTerms,
		})
	}
	return map[string]any{
		"filters": filters,
		"timeRange": map[string]any{
			"start": input.From, "end": input.To, "timeZone": "UTC", "granularity": input.Granularity,
		},
		"pagination": map[string]any{
			"offset": input.Offset, "pageSize": defaultInt(input.Limit, 100),
		},
	}
}

func changeHistoryPayload(input appleads.ChangeHistoryQuery) map[string]any {
	filters := []map[string]any{
		{"field": "eventTime", "operator": "BETWEEN", "value": []string{input.From, input.To}},
	}
	if len(input.EntityTypes) > 0 {
		filters = append(filters, map[string]any{
			"field": "entityType", "operator": "IN", "value": input.EntityTypes,
		})
	}
	if len(input.EventTypes) > 0 {
		filters = append(filters, map[string]any{
			"field": "eventType", "operator": "IN", "value": input.EventTypes,
		})
	}
	return map[string]any{
		"filters": filters,
		"sorting": []map[string]any{{"field": "eventTime", "order": "DESC"}},
		"pagination": map[string]any{
			"offset": input.Offset, "pageSize": defaultInt(input.Limit, 50),
		},
		"options": map[string]any{"needTotals": "true"},
	}
}

func normalizePopularityGenre(value string) string {
	value = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "&", " "))
	var out strings.Builder
	underscore := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			underscore = false
			continue
		}
		if out.Len() > 0 && !underscore {
			out.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(out.String(), "_")
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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

func parseIDOrToken(value string) any {
	var parsed int64
	if _, err := fmt.Sscan(value, &parsed); err == nil {
		return parsed
	}
	return value
}
