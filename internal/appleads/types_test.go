package appleads

import (
	"reflect"
	"testing"
)

func TestExtractCreatedResourceID(t *testing.T) {
	for _, tc := range []struct {
		response RawResponse
		want     string
	}{
		{RawResponse{"result": map[string]any{"id": float64(42)}}, "42"},
		{RawResponse{"data": map[string]any{"id": float64(43)}}, "43"},
		{RawResponse{"result": map[string]any{"id": float64(2145031798)}}, "2145031798"},
		{RawResponse{"id": "44"}, "44"},
		{RawResponse{"result": map[string]any{"id": nil}}, ""},
		{RawResponse{}, ""},
	} {
		if got := ExtractID(tc.response); got != tc.want {
			t.Fatalf("ExtractID(%v) = %q, want %q", tc.response, got, tc.want)
		}
	}
}

func TestMoneyNormalizationPreservesCallerInputAndText(t *testing.T) {
	for _, currency := range []string{"CNY", " cny ", "RMB", "USD"} {
		input := PlanCreateInput{AppID: "123", Countries: []string{"US"}, Currency: currency,
			Keywords: []KeywordPlan{{Text: " CNY exchange ", MatchType: "exact", Bid: Money{Amount: "5", Currency: currency}}}}
		before := input.Keywords[0]
		got, err := NormalizePlan(input)
		if err != nil {
			t.Fatal(err)
		}
		want := "RMB"
		if currency == "USD" {
			want = "USD"
		}
		if got.Currency != want || got.Keywords[0].Bid.Currency != want || got.Keywords[0].Text != "CNY exchange" {
			t.Fatalf("unexpected normalized plan: %+v", got)
		}
		if !reflect.DeepEqual(input.Keywords[0], before) {
			t.Fatal("caller input was mutated")
		}
	}
}
