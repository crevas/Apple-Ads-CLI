package cli

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
)

// Exercises the actual CLI, OAuth, headers, HTTP serialization and response
// decoding against a local Apple-shaped server; never contacts a live account.
func setupAppleMock(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			fmt.Fprint(w, `{"access_token":"test-token","expires_in":3600}`)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing OAuth bearer token")
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.p8")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"LILY_ADS_CONFIG":    filepath.Join(dir, "config.json"),
		"APPLE_ADS_PROVIDER": "", "LILY_ADS_PROVIDER": "",
		"APPLE_ADS_CLIENT_ID": "test-client", "APPLE_ADS_TEAM_ID": "test-team", "APPLE_ADS_KEY_ID": "test-key",
		"APPLE_ADS_PRIVATE_KEY_PATH": keyPath, "APPLE_ADS_AD_ACCOUNT_ID": "321", "APPLE_ADS_ORG_ID": "654",
		"APPLE_ADS_TOKEN_URL": server.URL + "/token", "APPLE_ADS_PLATFORM_BASE_URL": server.URL + "/v1",
		"APPLE_ADS_V5_BASE_URL": server.URL + "/v5",
	} {
		t.Setenv(name, value)
	}
}

func TestDefaultV1CampaignPackageOverHTTP(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		cpa, v1CPA, bulkFailure bool
		currency, status        string
		code                    int
	}{
		{name: "v1 CPA", cpa: true, v1CPA: true, currency: "CNY", status: "verified"},
		{name: "compatibility CPA", cpa: true, currency: "RMB", status: "verified"},
		{name: "mismatched CPA currency", cpa: true, currency: "USD", status: "mismatch", code: 1},
		{name: "no CPA needs no compatibility reads"},
		{name: "partial bulk failure", bulkFailure: true, code: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			var bodies []map[string]any
			setupAppleMock(t, func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				if strings.HasPrefix(r.URL.Path, "/v1/") && r.Header.Get("X-Ap-Context") != "adAccountId=321;" {
					t.Error("wrong v1 account scope")
				}
				var body map[string]any
				if r.Method == "POST" {
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					bodies = append(bodies, body)
				}
				switch r.Method + " " + r.URL.Path {
				case "POST /v1/campaigns":
					fmt.Fprint(w, `{"result":{"id":55}}`)
				case "POST /v1/adgroups":
					fmt.Fprint(w, `{"result":{"id":77,"campaignId":55}}`)
				case "POST /v1/keywords/bulk-create":
					if tc.bulkFailure {
						fmt.Fprint(w, `{"result":[{"correlationId":1,"success":false,"error":{"message":"invalid keyword"}}]}`)
					} else {
						fmt.Fprint(w, `{"result":[{"correlationId":1,"success":true,"result":{"id":88}}]}`)
					}
				case "POST /v1/negative-keywords/bulk-create":
					fmt.Fprint(w, `{"result":[{"correlationId":1,"success":true,"result":{"id":99}}]}`)
				case "GET /v1/adgroups/77":
					group := map[string]any{"id": 77, "campaignId": 55}
					if tc.v1CPA {
						group["cpaCap"] = map[string]any{"value": appleads.Money{Amount: "5.00", Currency: tc.currency}}
					}
					json.NewEncoder(w).Encode(map[string]any{"result": group})
				case "GET /v5/campaigns/55/adgroups/77":
					if r.Header.Get("X-Ap-Context") != "orgId=654" {
						t.Error("wrong v5 compatibility scope")
					}
					json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 77, "campaignId": 55, "cpaGoal": appleads.Money{Amount: "5", Currency: tc.currency}}})
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(404)
				}
			})
			args := []string{"ads", "plan", "create", "--app-id", "123456", "--country", "AU", "--name", "CNY test", "--currency", " cny ", "--daily-budget", "200", "--bid", "10", "--exact-keywords", "CNY exchange", "--campaign-negative-exact", "free", "--adgroup-negative-exact", "jobs", "--yes"}
			if tc.cpa {
				args = append(args, "--cpa-goal", "5")
			}
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), args, &stdout, &stderr)
			if code != tc.code {
				t.Fatalf("exit = %d, want %d: %s\n%s", code, tc.code, stderr.String(), stdout.String())
			}
			var result appleads.PlanCreateResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Provider != "platform" || result.Executed[0].ID != "55" || result.Executed[1].ID != "77" {
				t.Fatalf("missing created IDs: %+v", result)
			}
			wantRequests := []string{"POST /v1/campaigns", "POST /v1/adgroups", "POST /v1/keywords/bulk-create"}
			if !tc.bulkFailure {
				wantRequests = append(wantRequests, "POST /v1/negative-keywords/bulk-create", "POST /v1/negative-keywords/bulk-create")
			}
			if tc.cpa {
				wantRequests = append(wantRequests, "GET /v1/adgroups/77")
				if !tc.v1CPA {
					wantRequests = append(wantRequests, "GET /v5/campaigns/55/adgroups/77")
				}
				if result.CPAGoalVerification == nil || result.CPAGoalVerification.Status != tc.status {
					t.Fatalf("verification: %+v", result.CPAGoalVerification)
				}
			} else if result.CPAGoalVerification != nil {
				t.Fatal("unexpected CPA verification")
			}
			if !reflect.DeepEqual(requests, wantRequests) {
				t.Fatalf("requests = %v, want %v", requests, wantRequests)
			}
			if (result.Error != "") != (tc.code != 0) {
				t.Fatalf("error result: %+v", result)
			}
			if tc.code != 0 && len(result.NextActions) == 0 {
				t.Fatal("failed creation must explain retry risk")
			}
			campaign := bodies[0]
			assertJSON(t, campaign["bidStrategy"], map[string]any{"bidStrategyType": "MANUAL_CPT", "bidStrategyGoal": "TAP"})
			assertJSON(t, campaign["dailyBudget"], map[string]any{"value": appleads.Money{Amount: "200", Currency: "RMB"}})
			if campaign["name"] != "CNY test" || campaign["status"] != "PAUSED" {
				t.Fatal("campaign text or status changed")
			}
			assertJSON(t, bodies[1]["bidStrategy"], map[string]any{"bidStrategyType": "MANUAL_CPT", "bidStrategyGoal": "TAP", "bid": appleads.Money{Amount: "10", Currency: "RMB"}})
			if tc.cpa {
				assertJSON(t, bodies[1]["cpaCap"], map[string]any{"value": appleads.Money{Amount: "5", Currency: "RMB"}})
			}
			keyword := bodies[2]["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
			assertJSON(t, keyword["bid"], appleads.Money{Amount: "10", Currency: "RMB"})
			if keyword["text"] != "CNY exchange" {
				t.Fatal("keyword text changed")
			}
			if !tc.bulkFailure {
				campaignNegative := bodies[3]["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
				adGroupNegative := bodies[4]["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
				if campaignNegative["campaignId"] != float64(55) || campaignNegative["adGroupId"] != nil || adGroupNegative["adGroupId"] != float64(77) || adGroupNegative["campaignId"] != nil {
					t.Fatal("negative keyword scopes mixed")
				}
			}
		})
	}
}

func TestCampaignReportDefaultAndV5OverrideOverHTTP(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			setupAppleMock(t, func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if legacy {
					if r.URL.Path != "/v5/reports/campaigns" || r.Header.Get("X-Ap-Context") != "orgId=654" {
						t.Errorf("wrong legacy routing: %s", r.URL.Path)
					}
					fmt.Fprint(w, `{"data":[]}`)
				} else {
					if r.URL.Path != "/v1/reports/apps/campaigns/query" || r.Header.Get("X-Ap-Context") != "adAccountId=321;" {
						t.Errorf("wrong v1 routing: %s", r.URL.Path)
					}
					assertJSON(t, body["pagination"], map[string]any{"offset": 20, "pageSize": 7})
					assertJSON(t, body["filters"], []map[string]any{{"field": "promotedObjectId", "operator": "EQUALS", "value": "123456"}})
					assertJSON(t, body["options"], map[string]any{"includeRows": []string{"GRAND_TOTAL"}})
					fmt.Fprint(w, `{"result":[]}`)
				}
			})
			args := []string{"ads", "reports", "campaigns", "--app-id", "123456", "--limit", "7", "--offset", "20", "--from", "2026-11-01", "--to", "2026-11-02", "--no-revenue"}
			if legacy {
				args = append(args, "--provider", "v5")
			}
			var stdout, stderr bytes.Buffer
			if code := Run(context.Background(), args, &stdout, &stderr); code != 0 {
				t.Fatalf("exit %d: %s", code, stderr.String())
			}
		})
	}
}

func assertJSON(t *testing.T, actual, expected any) {
	t.Helper()
	a, err := json.Marshal(actual)
	if err != nil {
		t.Error(err)
		return
	}
	e, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
		return
	}
	if string(a) != string(e) {
		t.Errorf("got %s, want %s", a, e)
	}
}
