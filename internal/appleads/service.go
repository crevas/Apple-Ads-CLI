package appleads

import "fmt"

type PlanService struct {
	Provider Provider
	Client   RequestContext
}

func (s PlanService) Create(input PlanCreateInput) (PlanCreateResult, error) {
	normalized, err := NormalizePlan(input)
	if err != nil {
		return PlanCreateResult{}, err
	}

	result := PlanCreateResult{
		Tool:          "Apple Ads CLI by Lily",
		Provider:      s.Provider.Name(),
		Mode:          "dry-run",
		CorrelationID: normalized.CorrelationID,
		Planned:       s.Provider.PlannedRequests(normalized),
		NextActions: []string{
			"Review the planned Apple Ads write operations.",
			"Run again with --yes to execute the plan.",
		},
		SafetyReminder: "Write operations are dry-run by default for AI-agent safety.",
	}

	if !normalized.Execute {
		return result, nil
	}

	result.Mode = "execute"
	result.NextActions = nil

	campaignResp, campaignID, err := s.Provider.CreateCampaign(s.Client, CampaignCreate{
		Name:        normalized.Name,
		AppID:       normalized.AppID,
		Countries:   normalized.Countries,
		Currency:    normalized.Currency,
		DailyBudget: normalized.DailyBudget,
		StartTime:   normalized.StartTime,
		EndTime:     normalized.EndTime,
		Status:      normalized.Status,
		Supply:      normalized.Supply,
	})
	if err != nil {
		return result, fmt.Errorf("create campaign: %w", err)
	}
	result.Executed = append(result.Executed, ExecutedStep{Step: "create_campaign", ID: campaignID, Response: campaignResp})
	if campaignID == "" {
		return result, fmt.Errorf("create campaign response did not include id")
	}

	adGroupResp, adGroupID, err := s.Provider.CreateAdGroup(s.Client, AdGroupCreate{
		CampaignID: campaignID,
		Name:       normalized.AdGroupName,
		Currency:   normalized.Currency,
		Bid:        normalized.DefaultBid,
		CPAGoal:    normalized.CPAGoal,
		StartTime:  normalized.StartTime,
		EndTime:    normalized.EndTime,
		Status:     normalized.Status,
	})
	if err != nil {
		return result, fmt.Errorf("create ad group: %w", err)
	}
	result.Executed = append(result.Executed, ExecutedStep{Step: "create_ad_group", ID: adGroupID, Response: adGroupResp})
	if adGroupID == "" {
		return result, fmt.Errorf("create ad group response did not include id")
	}

	var keywords []KeywordCreate
	for _, keyword := range normalized.Keywords {
		keywords = append(keywords, KeywordCreate{
			CampaignID: campaignID,
			AdGroupID:  adGroupID,
			Text:       keyword.Text,
			MatchType:  keyword.MatchType,
			Currency:   normalized.Currency,
			Bid:        keyword.Bid.Amount,
			Status:     normalized.Status,
		})
	}
	keywordResp, err := s.Provider.BulkCreateKeywords(s.Client, keywords, normalized.AllowPartial)
	if err != nil {
		return result, fmt.Errorf("bulk create keywords: %w", err)
	}
	result.Executed = append(result.Executed, ExecutedStep{Step: "bulk_create_keywords", Response: keywordResp})

	if len(normalized.CampaignNegativeKeywords) > 0 {
		negatives := make([]NegativeKeywordCreate, 0, len(normalized.CampaignNegativeKeywords))
		for _, keyword := range normalized.CampaignNegativeKeywords {
			negatives = append(negatives, NegativeKeywordCreate{
				CampaignID: campaignID,
				Text:       keyword.Text,
				MatchType:  keyword.MatchType,
				Status:     normalized.Status,
			})
		}
		negativeResp, err := s.Provider.BulkCreateNegativeKeywords(s.Client, negatives, normalized.AllowPartial)
		if err != nil {
			return result, fmt.Errorf("bulk create campaign negative keywords: %w", err)
		}
		result.Executed = append(result.Executed, ExecutedStep{Step: "bulk_create_campaign_negative_keywords", Response: negativeResp})
	}

	if len(normalized.AdGroupNegativeKeywords) > 0 {
		negatives := make([]NegativeKeywordCreate, 0, len(normalized.AdGroupNegativeKeywords))
		for _, keyword := range normalized.AdGroupNegativeKeywords {
			negatives = append(negatives, NegativeKeywordCreate{
				CampaignID: campaignID,
				AdGroupID:  adGroupID,
				Text:       keyword.Text,
				MatchType:  keyword.MatchType,
				Status:     normalized.Status,
			})
		}
		negativeResp, err := s.Provider.BulkCreateNegativeKeywords(s.Client, negatives, normalized.AllowPartial)
		if err != nil {
			return result, fmt.Errorf("bulk create ad group negative keywords: %w", err)
		}
		result.Executed = append(result.Executed, ExecutedStep{Step: "bulk_create_adgroup_negative_keywords", Response: negativeResp})
	}

	if normalized.Creative.Kind != "" && normalized.Creative.Kind != "none" {
		creativeID := normalized.Creative.CreativeID
		if normalized.Creative.Kind == "default" || normalized.Creative.Kind == "product-page" {
			creativeResp, createdCreativeID, err := s.Provider.CreateCreative(s.Client, CreativeCreate{
				AppID:         normalized.AppID,
				ProductPageID: normalized.Creative.ProductPageID,
				Name:          defaultCreativeName(normalized),
			})
			if err != nil {
				return result, fmt.Errorf("create creative: %w", err)
			}
			creativeID = createdCreativeID
			result.Executed = append(result.Executed, ExecutedStep{Step: "create_creative", ID: creativeID, Response: creativeResp})
		}
		if creativeID == "" {
			return result, fmt.Errorf("creative id is required to attach an ad")
		}
		adResp, adID, err := s.Provider.CreateAd(s.Client, AdCreate{
			CampaignID: campaignID,
			AdGroupID:  adGroupID,
			CreativeID: creativeID,
			Name:       defaultAdName(normalized),
			Status:     normalized.Status,
		})
		if err != nil {
			return result, fmt.Errorf("create ad: %w", err)
		}
		result.Executed = append(result.Executed, ExecutedStep{Step: "create_ad", ID: adID, Response: adResp})
	}
	return result, nil
}

func defaultCreativeName(input PlanCreateInput) string {
	if input.Creative.Name != "" {
		return input.Creative.Name
	}
	return input.Name + " Creative"
}

func defaultAdName(input PlanCreateInput) string {
	if input.Creative.AdName != "" {
		return input.Creative.AdName
	}
	return input.Name + " Ad"
}
