package appleads

import "testing"

func TestPlanCreateDryRunIncludesReviewAndUserConfirmation(t *testing.T) {
	result, err := PlanService{Provider: fakePlanProvider{}}.Create(PlanCreateInput{
		Name:        "IELTS TW",
		AppID:       "734412264",
		Countries:   []string{"TW"},
		Currency:    "RMB",
		DailyBudget: "50",
		AdGroupName: "IELTS TW Search",
		DefaultBid:  "160",
		CPAGoal:     "10",
		Keywords: []KeywordPlan{
			{Text: "ielts", MatchType: "BROAD", Bid: Money{Amount: "160", Currency: "RMB"}},
			{Text: "ielts prep", MatchType: "EXACT", Bid: Money{Amount: "120", Currency: "RMB"}},
		},
		CampaignNegativeKeywords: []NegativeKeywordPlan{
			{Text: "jobs", MatchType: "EXACT"},
		},
		AdGroupNegativeKeywords: []NegativeKeywordPlan{
			{Text: "wallpaper", MatchType: "BROAD"},
		},
		Creative: CreativeSelection{Kind: "product-page", ProductPageID: "pp_123"},
		Status:   "ENABLED",
		Supply:   "APPSTORE_SEARCH_RESULTS",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if result.Mode != "dry-run" {
		t.Fatalf("mode = %q, want dry-run", result.Mode)
	}
	if result.Review.CampaignName != "IELTS TW" {
		t.Fatalf("campaignName = %q, want IELTS TW", result.Review.CampaignName)
	}
	if result.Review.DailyBudget.Amount != "50" || result.Review.DailyBudget.Currency != "RMB" {
		t.Fatalf("dailyBudget = %#v, want 50 RMB", result.Review.DailyBudget)
	}
	if result.Review.CPAGoal == nil || result.Review.CPAGoal.Amount != "10" {
		t.Fatalf("cpaGoal = %#v, want 10", result.Review.CPAGoal)
	}
	if result.Review.Keywords.Total != 2 || result.Review.Keywords.Broad != 1 || result.Review.Keywords.Exact != 1 {
		t.Fatalf("keywords = %#v, want total 2 broad 1 exact 1", result.Review.Keywords)
	}
	if result.Review.Negatives.Total != 2 || result.Review.Negatives.CampaignExact != 1 || result.Review.Negatives.AdGroupBroad != 1 {
		t.Fatalf("negative keywords = %#v, want total 2 campaign exact 1 ad group broad 1", result.Review.Negatives)
	}
	if result.Confirmation == nil {
		t.Fatal("confirmation is nil")
	}
	if got := len(result.Confirmation.Choices); got != 3 {
		t.Fatalf("confirmation choices = %d, want 3", got)
	}
	if result.Confirmation.Choices[0].ID != "confirm_create" {
		t.Fatalf("first confirmation choice = %q, want confirm_create", result.Confirmation.Choices[0].ID)
	}
	if result.Confirmation.Choices[0].RequiresWrite != true {
		t.Fatal("confirm_create should require writes")
	}
}

func TestPlanCreateDryRunAllowsDraftDefaultsButExecuteRequiresExplicitValues(t *testing.T) {
	result, err := PlanService{Provider: fakePlanProvider{}}.Create(PlanCreateInput{
		AppID:     "999999999",
		Countries: []string{"UK"},
	})
	if err != nil {
		t.Fatalf("Create dry-run returned error: %v", err)
	}
	if result.Review.CampaignName != "999999999-GB-Search-1" {
		t.Fatalf("campaignName = %q, want inferred name", result.Review.CampaignName)
	}
	if result.Review.DailyBudget.Amount != "20" {
		t.Fatalf("dailyBudget = %#v, want default 20", result.Review.DailyBudget)
	}
	if result.Review.DefaultBid.Amount != "1.50" {
		t.Fatalf("defaultBid = %#v, want default 1.50", result.Review.DefaultBid)
	}
	if result.Review.Status != "PAUSED" {
		t.Fatalf("status = %q, want PAUSED", result.Review.Status)
	}
	if len(result.Assumptions) == 0 {
		t.Fatal("assumptions should explain draft defaults")
	}

	_, err = PlanService{Provider: fakePlanProvider{}}.Create(PlanCreateInput{
		AppID:     "999999999",
		Countries: []string{"UK"},
		Execute:   true,
	})
	if err == nil {
		t.Fatal("Create execute returned nil error, want explicit value error")
	}
}

type fakePlanProvider struct{}

func (fakePlanProvider) Name() string { return "fake" }

func (fakePlanProvider) PlannedRequests(input PlanCreateInput) []PlannedRequest {
	return []PlannedRequest{{Step: "create_campaign", Method: "POST", Path: "/campaigns"}}
}

func (fakePlanProvider) CreateCampaign(ctx RequestContext, input CampaignCreate) (RawResponse, string, error) {
	return nil, "", nil
}

func (fakePlanProvider) CreateAdGroup(ctx RequestContext, input AdGroupCreate) (RawResponse, string, error) {
	return nil, "", nil
}

func (fakePlanProvider) BulkCreateKeywords(ctx RequestContext, keywords []KeywordCreate, allowPartial bool) (RawResponse, error) {
	return nil, nil
}

func (fakePlanProvider) BulkCreateNegativeKeywords(ctx RequestContext, keywords []NegativeKeywordCreate, allowPartial bool) (RawResponse, error) {
	return nil, nil
}

func (fakePlanProvider) CreateCreative(ctx RequestContext, input CreativeCreate) (RawResponse, string, error) {
	return nil, "", nil
}

func (fakePlanProvider) CreateAd(ctx RequestContext, input AdCreate) (RawResponse, string, error) {
	return nil, "", nil
}

func (fakePlanProvider) QueryCampaignReport(ctx RequestContext, input CampaignReportQuery) (RawResponse, error) {
	return nil, nil
}
