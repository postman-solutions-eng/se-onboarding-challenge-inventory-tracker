package httpapi

import (
	"net/http"
	"regexp"
	"strconv"

	"inventory-tracker-server/internal/model"
)

var templatePlaceholder = regexp.MustCompile(`^\{\{.+\}\}$`)

// checkAPIKey mirrors the mock's checkApiKey(): requires a non-empty,
// non-unresolved-template x-api-key header, and if the server was started
// with an API_KEY, requires an exact match. Returns false after already
// writing the 401 response.
func (a *API) checkAPIKey(w http.ResponseWriter, r *http.Request) bool {
	key := r.Header.Get("x-api-key")
	if key == "" || templatePlaceholder.MatchString(key) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "x-api-key header is required", nil)
		return false
	}
	if a.APIKey != "" && key != a.APIKey {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid API key", nil)
		return false
	}
	return true
}

// canned mirrors the mock's CANNED table for the x-mock-response-code
// backward-compat header.
var canned = map[string]model.Error{
	"400": {Code: "VALIDATION_ERROR", Message: "Request body is invalid", Details: []model.ErrorDetail{{Field: "field", Issue: "invalid value"}}},
	"401": {Code: "UNAUTHORIZED", Message: "x-api-key header is required or invalid", Details: []model.ErrorDetail{}},
	"404": {Code: "NOT_FOUND", Message: "The requested resource was not found", Details: []model.ErrorDetail{}},
	"409": {Code: "CONFLICT", Message: "A resource with this identifier already exists", Details: []model.ErrorDetail{}},
	"422": {Code: "INSUFFICIENT_STOCK", Message: "Issue quantity exceeds available stock", Details: []model.ErrorDetail{}},
}

// cannedOverride mirrors default.js's top-of-request-handler check for
// x-mock-response-code: if present and recognized, it short-circuits real
// routing/logic entirely (the 401 case skips the API-key check; others
// still require a valid key first).
func (a *API) cannedOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockCode := r.Header.Get("x-mock-response-code")
		if mockCode != "" {
			if body, ok := canned[mockCode]; ok {
				if mockCode != "401" && !a.checkAPIKey(w, r) {
					return
				}
				status, _ := strconv.Atoi(mockCode)
				writeJSON(w, status, body)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
