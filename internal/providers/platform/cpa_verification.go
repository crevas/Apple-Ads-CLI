package platform

import (
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

// VerifyCPAGoal checks persisted values, never the create request or response.
// v1 may omit deprecated cpaCap. Only that omission permits a legacy GET, using
// the explicitly configured org and the same campaign/ad group identities.
func (p Provider) VerifyCPAGoal(ctx appleads.RequestContext, campaignID, adGroupID string, expected appleads.Money) (appleads.CPAGoalVerification, error) {
	verification := appleads.CPAGoalVerification{Status: "unverified", Source: "v1"}
	response, err := ctx.Do("GET", "/adgroups/"+url.PathEscape(adGroupID), nil)
	if err != nil {
		return verification, fmt.Errorf("read CPA Goal from v1: %w", err)
	}
	group, err := verifiedAdGroup(response, "result", campaignID, adGroupID)
	if err != nil {
		return verification, err
	}
	goal, present := group["cpaCap"]
	if !present {
		if p.CPAGoalCompatibilityClient == nil {
			return verification, fmt.Errorf("CPA Goal verification unavailable: v1 omitted cpaCap; configure APPLE_ADS_ORG_ID for a read-only v5 compatibility check")
		}
		verification.Source = "v5-compatibility"
		response, err = p.CPAGoalCompatibilityClient.Do("GET", "/campaigns/"+url.PathEscape(campaignID)+"/adgroups/"+url.PathEscape(adGroupID), nil)
		if err != nil {
			return verification, fmt.Errorf("read CPA Goal from v5 compatibility endpoint: %w", err)
		}
		group, err = verifiedAdGroup(response, "data", campaignID, adGroupID)
		if err != nil {
			return verification, err
		}
		goal, present = group["cpaGoal"]
	} else if wrapper, ok := goal.(map[string]any); ok {
		goal, present = wrapper["value"]
	}
	if !present {
		return verification, fmt.Errorf("CPA Goal verification unavailable: response omitted the saved value")
	}
	if goal == nil {
		verification.Status = "mismatch"
		return verification, fmt.Errorf("CPA Goal mismatch: Apple returned an explicitly cleared value")
	}
	value, ok := goal.(map[string]any)
	if !ok {
		return verification, fmt.Errorf("CPA Goal verification unavailable: invalid Money response")
	}
	amount, _ := value["amount"].(string)
	currency, _ := value["currency"].(string)
	actualAmount := positiveDecimal(amount)
	expectedAmount := positiveDecimal(expected.Amount)
	if actualAmount == nil || expectedAmount == nil || strings.TrimSpace(currency) == "" {
		return verification, fmt.Errorf("CPA Goal verification unavailable: invalid amount or currency")
	}
	verification.Actual = &appleads.Money{Amount: amount, Currency: currency}
	if actualAmount.Cmp(expectedAmount) != 0 || appleads.NormalizeCurrency(currency) != appleads.NormalizeCurrency(expected.Currency) {
		verification.Status = "mismatch"
		return verification, fmt.Errorf("CPA Goal mismatch: requested %s %s, read back %s %s", expected.Amount, expected.Currency, amount, currency)
	}
	verification.Status = "verified"
	return verification, nil
}

func verifiedAdGroup(response appleads.RawResponse, field, campaignID, adGroupID string) (map[string]any, error) {
	group, ok := response[field].(map[string]any)
	if !ok || appleads.ExtractID(appleads.RawResponse(group)) != adGroupID || appleads.ExtractID(appleads.RawResponse{"id": group["campaignId"]}) != campaignID {
		return nil, fmt.Errorf("CPA Goal verification failed: ad group identity mismatch")
	}
	return group, nil
}

var decimalAmount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

func positiveDecimal(value string) *big.Rat {
	value = strings.TrimSpace(value)
	if !decimalAmount.MatchString(value) {
		return nil
	}
	amount, ok := new(big.Rat).SetString(value)
	if !ok || amount.Sign() <= 0 {
		return nil
	}
	return amount
}
