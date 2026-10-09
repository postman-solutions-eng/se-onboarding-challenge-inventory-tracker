# Inventory Tracker API — Workspace README

A Postman workspace for the **Inventory Tracker API**: a RESTful service for managing warehouse items, locations, stock levels, and stock movements.

---

## 📐 API Specification

**File:** `postman/specs/inventory-tracker.yaml`  
**Standard:** OpenAPI 3.0.3  
**Version:** 1.0.0

The spec describes a complete Inventory Tracker API with the following resource groups:

| Tag | Description |
| --- | --- |
| **Items** | Manage inventory items — SKU, name, unit, reorder threshold, and lifecycle status |
| **Locations** | Manage warehouse and bin locations where stock is held |
| **Stock** | Query current stock levels per item per location |
| **Movements** | Record and query stock movements (receive, issue, adjust) |
| **System** | System health and operational status |

### Endpoints at a Glance

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | System health check (no auth required) |
| GET | `/items` | List all items (paginated, filterable by status/search) |
| POST | `/items` | Create a new item |
| GET | `/items/{itemId}` | Get an item by ID |
| PATCH | `/items/{itemId}` | Partially update an item |
| DELETE | `/items/{itemId}` | Delete an item |
| GET | `/items/{itemId}/stock` | Get stock levels for an item |
| GET | `/locations` | List all locations |
| POST | `/locations` | Create a new location |
| GET | `/locations/{locationId}` | Get a location by ID |
| POST | `/stock-movements` | Record a stock movement (receive / issue / adjust) |
| GET | `/stock-movements` | List stock movements with optional filters |

**Authentication:** All endpoints (except `/health`) require an `x-api-key` header.

---

## 📦 Collection

**Name:** Inventory Tracker API Collection  
**Path:** `postman/collections/Inventory Tracker API Collection/`

The collection is organized into the following folders:

### Resource Folders (CRUD)

| Folder | Requests |
| --- | --- |
| `health` | System health check |
| `items` | List all items, Create a new item, Get by ID, Partially update, Delete |
| `items/{itemId}/stock` | Get stock level for an item |
| `locations` | List all locations, Create a new location, Get by ID |
| `stock-movements` | List movements (with filters), Record a movement |
| `auth` | 401 – No API key, 401 – Invalid API key |

### E2E Happy Path

An ordered 8-step flow that validates the full stock lifecycle end-to-end:

1. Create a location
    
2. Create an item
    
3. Receive 100 units (stock movement)
    
4. Check stock → assert 100
    
5. Issue 30 units (stock movement)
    
6. Check stock → assert 70
    
7. List movements by `itemId`
    
8. Delete item (cleanup)
    

### Failure Cases

Negative test scenarios covering all key error responses:

| File | Scenario |
| --- | --- |
| `400-invalid-quantity` | Stock movement with invalid quantity |
| `400-missing-field` | Create item/location with missing required field |
| `401-missing-token` | Request with no `x-api-key` header |
| `404-unknown-itemId` | Get/update/delete a non-existent item |
| `409-duplicate-sku` | Create item with a duplicate SKU |
| `409-duplicate-location-code` | Create location with a duplicate code |
| `422-issue-exceeds-stock` | Issue more units than available stock |

---

## 🌍 Environments

Three environments are provided, all sharing the same variable keys:

| Variable | Description |
| --- | --- |
| `baseUrl` | Base URL of the API server |
| `itemId` | Pre-seeded item UUID for tests |
| `locationId` | Pre-seeded primary location UUID |
| `secondLocationId` | Pre-seeded secondary location UUID |
| `movementId` | Populated at runtime by test scripts |
| `apiKey` | API key sent in the `x-api-key` header |

### Dev

**File:** `postman/environments/dev.environment.yaml`

- `baseUrl`: `http://localhost:4500`
    
- `apiKey`: `test-api-key-12345`
    
- Use when running the real server locally.
    

### Staging

**File:** `postman/environments/Staging.environment.yaml`

- `baseUrl`: `http://localhost:4500`
    
- `apiKey`: `test-api-key-12345`
    
- Points to the staging/shared deployment. Update `baseUrl` to your staging server URL.
    

> **⚠️ EKS 사용 시 주의:** Staging 환경을 EKS(Elastic Kubernetes Service)로 배포한 경우, 서비스가 **Cluster IP** 타입으로 구성되어 있다면 외부에서 직접 접근할 수 없습니다. 컬렉션 실행 전 아래 명령어로 포트 포워딩을 먼저 설정해야 합니다. 
  

``` bash
kubectl port-forward svc/<service-name> 4500:4500 -n <namespace>

 ```

> 포트 포워딩이 활성화된 상태에서 `baseUrl`의 `http://localhost:4500`으로 요청이 정상적으로 라우팅됩니다. 
  

### Local Mock

**File:** `postman/environments/Local Mock.environment.yaml`

- `baseUrl`: `http://localhost:4500`
    
- `apiKey`: `test-api-key-12345`
    
- Use together with the **Local Mock** server (see below) to run tests without a real backend.
    

---

## 🧪 Local Mock Server

**Path:** `postman/mocks/inventory-tracker-mock/`  
**Name:** Inventory Tracker Mock  
**Protocol:** HTTP · **Port:** `4500`

A fully dynamic, in-memory Node.js mock server that simulates the complete Inventory Tracker API — no real database required.

### Key Features

- **In-memory state** — items, locations, stock levels, and movements are stored in `Map` structures and live for the duration of the mock session.
    
- **Pre-seeded test data** — the server boots with well-known UUIDs that match the environment variables, so CRUD tests work out of the box without any setup requests.
    
- **API key validation** — enforces the `x-api-key` header; unset or template-variable (`{{apiKey}}`) keys return `401`.
    
- **Canned error overrides** — sending the `x-mock-response-code` header forces a specific error response (400 / 401 / 404 / 409 / 422) for backward compatibility with simple negative tests.
    
- **Pagination** — list endpoints support `page` and `pageSize` query parameters.
    
- **Auto-restart on change** — the mock restarts automatically when `default.js` is edited.
    
- **Verbose logging** — request/response bodies are logged to console.
    

### Files

| File | Description |
| --- | --- |
| `config.yaml` | Mock configuration (port, protocol, logging, collection association) |
| `default.js` | Default scenario handler — full routing + business logic |

### Seeded UUIDs

| Variable | UUID |
| --- | --- |
| `itemId` | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| `locationId` | `b2c3d4e5-f6a7-8901-bcde-f12345678901` |
| `secondLocationId` | `e5f6a7b8-c9d0-1234-ef01-345678901234` |

### Running the Mock

1. Select the **Local Mock** environment in Postman.
    
2. Start the mock server from the Postman sidebar (Mocks → Inventory Tracker Mock → Start).
    
3. Run the collection or individual requests — all calls resolve to `http://localhost:4500`.