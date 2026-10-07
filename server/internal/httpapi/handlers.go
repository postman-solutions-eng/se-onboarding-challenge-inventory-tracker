package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"inventory-tracker-server/internal/model"
	"inventory-tracker-server/internal/store"
)

// ── Pagination ───────────────────────────────────────────────────────────

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// paginate mirrors the mock's paginate(): Math.max(1, parsed) on both page
// and pageSize, no upper bound on pageSize (the spec's max:100 isn't
// enforced by the mock, so it isn't enforced here either).
func paginate(all []store.Entity, r *http.Request) model.ListResponse[store.Entity] {
	page := queryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}
	pageSize := queryInt(r, "pageSize", 20)
	if pageSize < 1 {
		pageSize = 1
	}

	start := (page - 1) * pageSize
	data := []store.Entity{}
	if start < len(all) {
		end := start + pageSize
		if end > len(all) {
			end = len(all)
		}
		data = all[start:end]
	}

	return model.ListResponse[store.Entity]{
		Data:       data,
		Pagination: model.Pagination{Page: page, PageSize: pageSize, Total: len(all)},
	}
}

// ── Health ───────────────────────────────────────────────────────────────

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, model.Health{Status: "ok"})
}

// ── Locations ────────────────────────────────────────────────────────────

func (a *API) handleListLocations(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, paginate(a.Store.ListLocations(), r))
}

func (a *API) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	b := readJSONBody(r)

	var miss []model.ErrorDetail
	if falsy(b["code"]) {
		miss = append(miss, model.ErrorDetail{Field: "code", Issue: "required field is missing"})
	}
	if falsy(b["name"]) {
		miss = append(miss, model.ErrorDetail{Field: "name", Issue: "required field is missing"})
	}
	if len(miss) > 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request body is missing required fields", miss)
		return
	}

	code := getString(b["code"])
	loc, conflict := a.Store.CreateLocation(store.Entity{
		"code": code, "name": getString(b["name"]), "description": stringOrDefault(b["description"], ""),
	})
	if conflict {
		writeError(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("A location with code %s already exists", code),
			[]model.ErrorDetail{{Field: "code", Issue: "Location code must be unique"}})
		return
	}
	writeJSON(w, http.StatusCreated, loc)
}

func (a *API) handleGetLocation(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	id := r.PathValue("locationId")
	loc, ok := a.Store.GetLocation(id)
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Location %s was not found", id), nil)
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

// ── Items ────────────────────────────────────────────────────────────────

func (a *API) handleListItems(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, paginate(a.Store.ListItems(), r))
}

func (a *API) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	b := readJSONBody(r)

	var miss []model.ErrorDetail
	if falsy(b["sku"]) {
		miss = append(miss, model.ErrorDetail{Field: "sku", Issue: "required field is missing"})
	}
	if falsy(b["name"]) {
		miss = append(miss, model.ErrorDetail{Field: "name", Issue: "required field is missing"})
	}
	if falsy(b["unit"]) {
		miss = append(miss, model.ErrorDetail{Field: "unit", Issue: "required field is missing"})
	}
	if len(miss) > 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request body is missing required fields", miss)
		return
	}

	sku := getString(b["sku"])
	item, conflict := a.Store.CreateItem(store.Entity{
		"sku": sku, "name": getString(b["name"]), "description": stringOrDefault(b["description"], ""),
		"unit": getString(b["unit"]), "reorderThreshold": intNullish(b, "reorderThreshold", 0),
		"status": stringOrDefault(b["status"], "active"),
	})
	if conflict {
		writeError(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("An item with SKU %s already exists", sku),
			[]model.ErrorDetail{{Field: "sku", Issue: "SKU must be unique across all items"}})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) handleGetItemStock(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	itemID := r.PathValue("itemId")
	writeJSON(w, http.StatusOK, paginate(a.Store.StockRowsForItem(itemID), r))
}

func (a *API) handleGetItem(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	id := r.PathValue("itemId")
	item, ok := a.Store.GetItem(id)
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Item %s was not found", id), nil)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	id := r.PathValue("itemId")
	b := readJSONBody(r)
	updated, ok := a.Store.UpdateItem(id, b)
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Item %s was not found", id), nil)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *API) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	id := r.PathValue("itemId")
	if !a.Store.DeleteItem(id) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Item %s was not found", id), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Stock movements ──────────────────────────────────────────────────────

func (a *API) handleListStockMovements(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	fItem := r.URL.Query().Get("itemId")
	fLoc := r.URL.Query().Get("locationId")
	fType := r.URL.Query().Get("type")

	list := a.Store.ListMovements()
	filtered := make([]store.Entity, 0, len(list))
	for _, mv := range list {
		if fItem != "" && mv["itemId"] != fItem {
			continue
		}
		if fLoc != "" && mv["locationId"] != fLoc {
			continue
		}
		if fType != "" && mv["type"] != fType {
			continue
		}
		filtered = append(filtered, mv)
	}
	writeJSON(w, http.StatusOK, paginate(filtered, r))
}

func (a *API) handleCreateStockMovement(w http.ResponseWriter, r *http.Request) {
	if !a.checkAPIKey(w, r) {
		return
	}
	b := readJSONBody(r)

	// Required-field presence (quantity uses a nullish check like the mock's
	// `b.quantity == null`, not a falsy check, so quantity: 0 reaches the
	// range check below instead of being reported as "missing").
	var miss []model.ErrorDetail
	if falsy(b["type"]) {
		miss = append(miss, model.ErrorDetail{Field: "type", Issue: "required field is missing"})
	}
	if falsy(b["itemId"]) {
		miss = append(miss, model.ErrorDetail{Field: "itemId", Issue: "required field is missing"})
	}
	if falsy(b["locationId"]) {
		miss = append(miss, model.ErrorDetail{Field: "locationId", Issue: "required field is missing"})
	}
	if isNullish(b, "quantity") {
		miss = append(miss, model.ErrorDetail{Field: "quantity", Issue: "required field is missing"})
	}
	if len(miss) > 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request body is missing required fields", miss)
		return
	}

	rawQuantity := b["quantity"]
	qf, isNum := rawQuantity.(float64)
	if !isNum || qf != math.Trunc(qf) || qf < 1 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "quantity must be a positive integer (minimum 1)",
			[]model.ErrorDetail{{Field: "quantity", Issue: fmt.Sprintf("received %v, must be >= 1", rawQuantity)}})
		return
	}
	quantity := int(qf)

	kind := getString(b["type"])
	itemID := getString(b["itemId"])
	locationID := getString(b["locationId"])
	delta := 0
	if df, ok := b["delta"].(float64); ok {
		delta = int(df)
	}
	referenceNumber := stringPtrOrNil(b["referenceNumber"])

	mv, insufficient, available := a.Store.RecordMovement(kind, itemID, locationID, quantity, delta, referenceNumber)
	if insufficient {
		writeError(w, http.StatusUnprocessableEntity, "INSUFFICIENT_STOCK",
			fmt.Sprintf("Cannot issue %d units; only %d available", quantity, available),
			[]model.ErrorDetail{{Field: "quantity", Issue: fmt.Sprintf("requested %d, available %d", quantity, available)}})
		return
	}
	writeJSON(w, http.StatusCreated, mv)
}
