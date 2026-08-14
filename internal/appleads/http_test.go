package appleads

import (
	"strings"
	"testing"
)

func TestAPIErrorPrefersPlatformValidationDetail(t *testing.T) {
	err := APIError{
		StatusCode: 400,
		Body: map[string]any{
			"error": map[string]any{
				"message": "One or more validation errors occurred",
				"details": []any{map[string]any{
					"message": "entityType filter is required",
				}},
			},
		},
	}

	if got := err.Error(); !strings.Contains(got, "entityType filter is required") {
		t.Fatalf("Error() = %q, want validation detail", got)
	}
}

func TestSuccessfulStatusEnvelopeWithErrorIsRejected(t *testing.T) {
	body := map[string]any{
		"error": map[string]any{
			"message": "One or more validation errors occurred",
			"details": []any{map[string]any{
				"message": "entityType filter is required",
			}},
		},
		"result": nil,
	}
	if !hasPlatformError(body) {
		t.Fatal("hasPlatformError = false, want true")
	}
	if got := (APIError{StatusCode: 200, Body: body}).Error(); !strings.HasPrefix(got, "apple ads api response error:") {
		t.Fatalf("Error() = %q, want response-error prefix", got)
	}
	if hasPlatformError(map[string]any{"error": nil, "result": []any{}}) {
		t.Fatal("hasPlatformError accepted a successful response")
	}
}

func TestNonJSONAPIErrorStillReportsHTTPStatus(t *testing.T) {
	got := (APIError{StatusCode: 503, Raw: "Service Unavailable"}).Error()
	if got != "apple ads api error 503: Service Unavailable" {
		t.Fatalf("Error() = %q", got)
	}
}
