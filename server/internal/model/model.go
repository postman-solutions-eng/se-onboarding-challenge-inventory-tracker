// Package model holds the response envelope shapes shared across every
// endpoint (pagination wrapper, error body) that aren't tied to a specific
// resource's dynamic fields.
package model

// Pagination mirrors the {page, pageSize, total} envelope returned
// alongside every list response.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

// ListResponse wraps a slice of items with a Pagination envelope, matching
// the mock's {data, pagination} response shape for every list endpoint.
type ListResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ErrorDetail is one entry in an Error's details array.
type ErrorDetail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// Error is the shared error body shape used by every non-2xx response.
type Error struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

// Health is the body returned by GET /health.
type Health struct {
	Status string `json:"status"`
}
