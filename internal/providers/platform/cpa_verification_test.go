package platform

import (
	"fmt"
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

type cpaContext struct {
	response appleads.RawResponse
	err      error
	calls    int
	path     string
}

func (c *cpaContext) Do(method, path string, body any) (appleads.RawResponse, error) {
	c.calls++
	c.path = path
	if method != "GET" || body != nil {
		return nil, fmt.Errorf("unexpected write")
	}
	return c.response, c.err
}

func TestCPAGoalReadback(t *testing.T) {
	money := func(amount, currency string) map[string]any {
		return map[string]any{"amount": amount, "currency": currency}
	}
	for _, tc := range []struct {
		name                                              string
		v1Goal                                            any
		omit, noLegacy, wrongV1, wrongLegacy, legacyError bool
		legacyGoal                                        any
		status, source                                    string
		legacyCalls                                       int
	}{
		{name: "v1 value", v1Goal: map[string]any{"value": money("5.00", "CNY")}, status: "verified", source: "v1"},
		{name: "v1 mismatch", v1Goal: map[string]any{"value": money("6", "RMB")}, status: "mismatch", source: "v1"},
		{name: "explicit null", status: "mismatch", source: "v1"},
		{name: "invalid wrapper", v1Goal: map[string]any{}, status: "unverified", source: "v1"},
		{name: "missing legacy scope", omit: true, noLegacy: true, status: "unverified", source: "v1"},
		{name: "legacy read", omit: true, legacyGoal: money("5.0", "RMB"), status: "verified", source: "v5-compatibility", legacyCalls: 1},
		{name: "legacy mismatch", omit: true, legacyGoal: money("4", "RMB"), status: "mismatch", source: "v5-compatibility", legacyCalls: 1},
		{name: "wrong currency", omit: true, legacyGoal: money("5", "USD"), status: "mismatch", source: "v5-compatibility", legacyCalls: 1},
		{name: "invalid amount", omit: true, legacyGoal: money("NaN", "RMB"), status: "unverified", source: "v5-compatibility", legacyCalls: 1},
		{name: "legacy missing value", omit: true, status: "unverified", source: "v5-compatibility", legacyCalls: 1},
		{name: "wrong v1 identity", omit: true, wrongV1: true, status: "unverified", source: "v1"},
		{name: "wrong legacy identity", omit: true, wrongLegacy: true, legacyGoal: money("5", "RMB"), status: "unverified", source: "v5-compatibility", legacyCalls: 1},
		{name: "legacy unavailable", omit: true, legacyError: true, status: "unverified", source: "v5-compatibility", legacyCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v1Group := map[string]any{"id": float64(2152472322), "campaignId": float64(2145031798)}
			if !tc.omit {
				v1Group["cpaCap"] = tc.v1Goal
			}
			if tc.wrongV1 {
				v1Group["campaignId"] = float64(99)
			}
			legacyGroup := map[string]any{"id": float64(2152472322), "campaignId": float64(2145031798)}
			if tc.legacyGoal != nil {
				legacyGroup["cpaGoal"] = tc.legacyGoal
			}
			if tc.wrongLegacy {
				legacyGroup["id"] = float64(99)
			}
			v1 := &cpaContext{response: appleads.RawResponse{"result": v1Group}}
			legacy := &cpaContext{response: appleads.RawResponse{"data": legacyGroup}}
			if tc.legacyError {
				legacy.err = fmt.Errorf("unavailable")
			}
			provider := New("321")
			if !tc.noLegacy {
				provider.CPAGoalCompatibilityClient = legacy
			}
			got, err := provider.VerifyCPAGoal(v1, "2145031798", "2152472322", appleads.Money{Amount: "5", Currency: "RMB"})
			if got.Status != tc.status || got.Source != tc.source || (err == nil) != (tc.status == "verified") {
				t.Fatalf("verification = %+v, error = %v", got, err)
			}
			if legacy.calls != tc.legacyCalls || v1.calls != 1 || v1.path != "/adgroups/2152472322" {
				t.Fatalf("unexpected read routing: v1=%+v legacy=%+v", v1, legacy)
			}
			if legacy.calls > 0 && legacy.path != "/campaigns/2145031798/adgroups/2152472322" {
				t.Fatal(legacy.path)
			}
		})
	}
}
