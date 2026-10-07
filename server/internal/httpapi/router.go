package httpapi

import "net/http"

// Routes wires every operation and wraps the whole mux with the
// x-mock-response-code backward-compat override, matching default.js's
// check-before-routing order.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.handleHealth)

	mux.HandleFunc("GET /locations", a.handleListLocations)
	mux.HandleFunc("POST /locations", a.handleCreateLocation)
	mux.HandleFunc("GET /locations/{locationId}", a.handleGetLocation)

	mux.HandleFunc("GET /items", a.handleListItems)
	mux.HandleFunc("POST /items", a.handleCreateItem)
	mux.HandleFunc("GET /items/{itemId}/stock", a.handleGetItemStock)
	mux.HandleFunc("GET /items/{itemId}", a.handleGetItem)
	mux.HandleFunc("PATCH /items/{itemId}", a.handleUpdateItem)
	mux.HandleFunc("DELETE /items/{itemId}", a.handleDeleteItem)

	mux.HandleFunc("GET /stock-movements", a.handleListStockMovements)
	mux.HandleFunc("POST /stock-movements", a.handleCreateStockMovement)

	mux.HandleFunc("/", a.handleNotFound)

	return a.cannedOverride(mux)
}

func (a *API) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint not defined", nil)
}
