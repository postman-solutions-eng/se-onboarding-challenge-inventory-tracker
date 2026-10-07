// Package store is the in-memory backing state for the API, reset every
// time the process restarts — the same lifecycle as the Node mock
// (postman/mocks/inventory-tracker-mock/default.js) it replaces.
package store

import (
	"strings"
	"sync"
)

// Entity is a loosely-typed JSON object. Items/locations/movements are
// stored this way (rather than as fixed structs) because the mock's
// PATCH /items/{id} does an object-spread merge of arbitrary body keys
// over the existing record — replicating that in Go is trivial with maps
// and would require reflection gymnastics with fixed structs.
type Entity map[string]any

// Well-known seed IDs, matching default.js exactly so the existing
// Postman environment's itemId/locationId/secondLocationId variables keep
// resolving to real seeded resources.
const (
	SeedItemID      = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	SeedLocationID  = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	SeedLocation2ID = "e5f6a7b8-c9d0-1234-ef01-345678901234"
)

type Store struct {
	mu sync.Mutex

	items     map[string]Entity
	itemOrder []string

	locations     map[string]Entity
	locationOrder []string

	stock      map[string]int // key: itemID+"|"+locationID
	stockOrder []string

	movements []Entity
}

func New() *Store {
	s := &Store{
		items:     map[string]Entity{},
		locations: map[string]Entity{},
		stock:     map[string]int{},
	}
	s.seed()
	return s
}

func (s *Store) seed() {
	s.items[SeedItemID] = Entity{
		"id": SeedItemID, "sku": "WIDGET-001", "name": "Blue Widget",
		"description": "A high-quality blue widget", "unit": "each",
		"reorderThreshold": 10, "status": "active",
		"createdAt": "2025-01-15T09:00:00Z", "updatedAt": "2025-06-01T14:30:00Z",
	}
	s.itemOrder = append(s.itemOrder, SeedItemID)

	s.locations[SeedLocationID] = Entity{
		"id": SeedLocationID, "code": "WH-EAST", "name": "East Warehouse",
		"description": "Main receiving warehouse", "createdAt": "2025-01-10T08:00:00Z",
	}
	s.locationOrder = append(s.locationOrder, SeedLocationID)

	s.locations[SeedLocation2ID] = Entity{
		"id": SeedLocation2ID, "code": "WH-WEST", "name": "West Warehouse",
		"description": "Secondary storage facility", "createdAt": "2025-03-05T07:00:00Z",
	}
	s.locationOrder = append(s.locationOrder, SeedLocation2ID)
}

// ── Items ────────────────────────────────────────────────────────────────

func (s *Store) ListItems() []Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entity, 0, len(s.itemOrder))
	for _, id := range s.itemOrder {
		out = append(out, s.items[id])
	}
	return out
}

func (s *Store) GetItem(id string) (Entity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[id]
	return e, ok
}

// CreateItem assigns id/createdAt/updatedAt and stores fields, returning
// conflict=true (and no entity) if the SKU is already taken.
func (s *Store) CreateItem(fields Entity) (Entity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sku := fields["sku"]
	for _, id := range s.itemOrder {
		if s.items[id]["sku"] == sku {
			return nil, true
		}
	}
	id := newUUID()
	ts := nowISO()
	item := Entity{
		"id": id, "sku": fields["sku"], "name": fields["name"],
		"description": fields["description"], "unit": fields["unit"],
		"reorderThreshold": fields["reorderThreshold"], "status": fields["status"],
		"createdAt": ts, "updatedAt": ts,
	}
	s.items[id] = item
	s.itemOrder = append(s.itemOrder, id)
	return item, false
}

// UpdateItem spreads patch over the existing record (any key in patch wins),
// then forces id/updatedAt — mirrors `{...item, ...b, id, updatedAt: now()}`.
func (s *Store) UpdateItem(id string, patch map[string]any) (Entity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.items[id]
	if !ok {
		return nil, false
	}
	updated := Entity{}
	for k, v := range existing {
		updated[k] = v
	}
	for k, v := range patch {
		updated[k] = v
	}
	updated["id"] = id
	updated["updatedAt"] = nowISO()
	s.items[id] = updated
	return updated, true
}

func (s *Store) DeleteItem(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	for i, v := range s.itemOrder {
		if v == id {
			s.itemOrder = append(s.itemOrder[:i], s.itemOrder[i+1:]...)
			break
		}
	}
	return true
}

// ── Locations ────────────────────────────────────────────────────────────

func (s *Store) ListLocations() []Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entity, 0, len(s.locationOrder))
	for _, id := range s.locationOrder {
		out = append(out, s.locations[id])
	}
	return out
}

func (s *Store) GetLocation(id string) (Entity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.locations[id]
	return e, ok
}

func (s *Store) CreateLocation(fields Entity) (Entity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code := fields["code"]
	for _, id := range s.locationOrder {
		if s.locations[id]["code"] == code {
			return nil, true
		}
	}
	id := newUUID()
	loc := Entity{
		"id": id, "code": fields["code"], "name": fields["name"],
		"description": fields["description"], "createdAt": nowISO(),
	}
	s.locations[id] = loc
	s.locationOrder = append(s.locationOrder, id)
	return loc, false
}

// ── Stock levels & movements ────────────────────────────────────────────

func stockKey(itemID, locationID string) string { return itemID + "|" + locationID }

// StockRowsForItem returns stock rows for itemID in first-insertion order,
// with updatedAt generated fresh at read time (the mock never persists a
// per-row timestamp either).
func (s *Store) StockRowsForItem(itemID string) []Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Entity{}
	for _, key := range s.stockOrder {
		parts := strings.SplitN(key, "|", 2)
		if parts[0] == itemID {
			out = append(out, Entity{
				"itemId": itemID, "locationId": parts[1],
				"quantity": s.stock[key], "updatedAt": nowISO(),
			})
		}
	}
	return out
}

func (s *Store) ListMovements() []Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entity, len(s.movements))
	copy(out, s.movements)
	return out
}

// RecordMovement atomically checks stock sufficiency (for issue) and
// applies the movement, so a check-then-act race can't occur under
// concurrent requests — the Node mock gets this atomicity for free from
// being single-threaded; this is the Go equivalent for the same observable
// behavior under the collection runner's sequential requests.
func (s *Store) RecordMovement(kind, itemID, locationID string, quantity, delta int, referenceNumber *string) (movement Entity, insufficient bool, available int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := stockKey(itemID, locationID)
	current := s.stock[key] // zero value if the key was never set, same as JS Map.get() || 0

	if kind == "issue" && quantity > current {
		return nil, true, current
	}

	var next int
	switch kind {
	case "receive":
		next = current + quantity
	case "issue":
		next = current - quantity
	case "adjust":
		next = current + delta
		if next < 0 {
			next = 0
		}
	}
	// Only register the key once a movement actually sets it, mirroring
	// the mock where Map.set() is never called on the insufficient-stock
	// failure path above (so a failed issue leaves no stock row behind).
	if _, existed := s.stock[key]; !existed {
		s.stockOrder = append(s.stockOrder, key)
	}
	s.stock[key] = next

	mv := Entity{
		"id": newUUID(), "type": kind, "itemId": itemID, "locationId": locationID,
		"quantity": quantity, "referenceNumber": referenceNumber, "createdAt": nowISO(),
	}
	s.movements = append(s.movements, mv)
	return mv, false, 0
}
