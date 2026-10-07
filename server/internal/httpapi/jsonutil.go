package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"inventory-tracker-server/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string, details []model.ErrorDetail) {
	if details == nil {
		details = []model.ErrorDetail{}
	}
	writeJSON(w, status, model.Error{Code: code, Message: message, Details: details})
}

// readJSONBody mirrors the mock's getBody(): any read/parse failure or an
// empty body resolves to an empty object rather than an error.
func readJSONBody(r *http.Request) map[string]any {
	body := map[string]any{}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		return body
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return map[string]any{}
	}
	return body
}

// truthy/falsy mirror JavaScript's loose truthiness for the field types
// that appear in request bodies here (strings, bools, numbers, nil).
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	default:
		return true
	}
}

func falsy(v any) bool { return !truthy(v) }

// isNullish mirrors `v == null` in JS: true only when the key is absent or
// explicitly JSON null (unlike falsy, it does NOT treat 0/"" as missing).
func isNullish(m map[string]any, key string) bool {
	v, ok := m[key]
	return !ok || v == nil
}

func getString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// stringOrDefault mirrors `v || def`.
func stringOrDefault(v any, def string) string {
	if falsy(v) {
		return def
	}
	return getString(v)
}

// intNullish mirrors `v ?? def`: only a missing/null value falls back to
// def; an explicit 0 is kept.
func intNullish(m map[string]any, key string, def int) int {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	if f, ok2 := v.(float64); ok2 {
		return int(f)
	}
	return def
}

// stringPtrOrNil mirrors `v || null`.
func stringPtrOrNil(v any) *string {
	if falsy(v) {
		return nil
	}
	s := getString(v)
	return &s
}
