// Package httpapi implements the HTTP surface of the Inventory Tracker API,
// ported line-for-line from postman/mocks/inventory-tracker-mock/default.js
// so the existing Postman collection passes unchanged against it.
package httpapi

import "inventory-tracker-server/internal/store"

type API struct {
	Store *store.Store
	// APIKey, when non-empty, is the required exact value of the
	// x-api-key header (mirrors the mock's VALID_API_KEY / API_KEY env var).
	APIKey string
}

func New(s *store.Store, apiKey string) *API {
	return &API{Store: s, APIKey: apiKey}
}
