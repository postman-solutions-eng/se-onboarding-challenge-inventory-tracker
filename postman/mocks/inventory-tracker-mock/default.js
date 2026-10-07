/**
 * Inventory Tracker – dynamic in-memory mock (v2)
 * - In-memory state for items, locations, stock, movements
 * - Backward-compat: x-mock-response-code header overrides real logic
 * - Pre-seeded with well-known UUIDs used by the original CRUD tests
 */
const http = require('http');
const PORT = process.env.PORT || 4500;
const VALID_API_KEY = process.env.API_KEY || null;

// ── In-memory state ───────────────────────────────────────────────────────────
const items     = new Map();
const locations = new Map();
const stock     = new Map();   // `${itemId}|${locationId}` → qty
const movements = [];

// ── Seed well-known test IDs (used by original CRUD folder tests) ─────────────
const SEED_ITEM_ID      = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
const SEED_LOCATION_ID  = 'b2c3d4e5-f6a7-8901-bcde-f12345678901';
const SEED_LOCATION2_ID = 'e5f6a7b8-c9d0-1234-ef01-345678901234';

items.set(SEED_ITEM_ID, {
  id: SEED_ITEM_ID, sku: 'WIDGET-001', name: 'Blue Widget',
  description: 'A high-quality blue widget', unit: 'each',
  reorderThreshold: 10, status: 'active',
  createdAt: '2025-01-15T09:00:00Z', updatedAt: '2025-06-01T14:30:00Z'
});
locations.set(SEED_LOCATION_ID, {
  id: SEED_LOCATION_ID, code: 'WH-EAST', name: 'East Warehouse',
  description: 'Main receiving warehouse', createdAt: '2025-01-10T08:00:00Z'
});
locations.set(SEED_LOCATION2_ID, {
  id: SEED_LOCATION2_ID, code: 'WH-WEST', name: 'West Warehouse',
  description: 'Secondary storage facility', createdAt: '2025-03-05T07:00:00Z'
});

// ── Helpers ───────────────────────────────────────────────────────────────────
function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}
function now() { return new Date().toISOString(); }
function getBody(req) {
  return new Promise(resolve => {
    let raw = '';
    req.on('data', c => { raw += c; });
    req.on('end', () => { try { resolve(raw ? JSON.parse(raw) : {}); } catch (_) { resolve({}); } });
    req.on('error', () => resolve({}));
  });
}
function send(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}
function checkApiKey(req, res) {
  const key = req.headers['x-api-key'];
  if (!key || /^\{\{.+\}\}$/.test(key)) {
    send(res, 401, { code: 'UNAUTHORIZED', message: 'x-api-key header is required', details: [] });
    return false;
  }
  if (VALID_API_KEY && key !== VALID_API_KEY) {
    send(res, 401, { code: 'UNAUTHORIZED', message: 'Invalid API key', details: [] });
    return false;
  }
  return true;
}
function paginate(arr, url) {
  const page     = Math.max(1, parseInt(url.searchParams.get('page')     || '1',  10));
  const pageSize = Math.max(1, parseInt(url.searchParams.get('pageSize') || '20', 10));
  const start    = (page - 1) * pageSize;
  return { data: arr.slice(start, start + pageSize), pagination: { page, pageSize, total: arr.length } };
}

// Canned error responses for x-mock-response-code header (backward compat)
const CANNED = {
  '400': { code: 'VALIDATION_ERROR',   message: 'Request body is invalid',                       details: [{ field: 'field', issue: 'invalid value' }] },
  '401': { code: 'UNAUTHORIZED',       message: 'x-api-key header is required or invalid',       details: [] },
  '404': { code: 'NOT_FOUND',          message: 'The requested resource was not found',           details: [] },
  '409': { code: 'CONFLICT',           message: 'A resource with this identifier already exists', details: [] },
  '422': { code: 'INSUFFICIENT_STOCK', message: 'Issue quantity exceeds available stock',         details: [] },
};

// ── Server ────────────────────────────────────────────────────────────────────
const server = http.createServer(async (req, res) => {
  const url    = new URL(req.url || '/', 'http://localhost');
  const path   = url.pathname;
  const method = req.method;

  // x-mock-response-code: honour canned override (backward compat with old tests)
  const mockCode = req.headers['x-mock-response-code'];
  if (mockCode && CANNED[mockCode]) {
    if (mockCode !== '401' && !checkApiKey(req, res)) return;
    return send(res, parseInt(mockCode, 10), CANNED[mockCode]);
  }

  // Pre-compute path matches
  const mLocId  = path.match(/^\/locations\/([^/]+)$/);
  const mStock  = path.match(/^\/items\/([^/]+)\/stock$/);
  const mItemId = path.match(/^\/items\/([^/]+)$/);

  // GET /health ─────────────────────────────────────────────────────────────────
  // @endpoint GET /health
  if (method === 'GET' && path === '/health') return send(res, 200, { status: 'ok' });

  // LOCATIONS ───────────────────────────────────────────────────────────────────
  // @endpoint GET /locations
  if (method === 'GET' && path === '/locations') {
    if (!checkApiKey(req, res)) return;
    return send(res, 200, paginate([...locations.values()], url));
  }
  // @endpoint POST /locations
  if (method === 'POST' && path === '/locations') {
    if (!checkApiKey(req, res)) return;
    const b = await getBody(req);
    const miss = [];
    if (!b.code) miss.push({ field: 'code', issue: 'required field is missing' });
    if (!b.name) miss.push({ field: 'name', issue: 'required field is missing' });
    if (miss.length) return send(res, 400, { code: 'VALIDATION_ERROR', message: 'Request body is missing required fields', details: miss });
    if ([...locations.values()].some(l => l.code === b.code))
      return send(res, 409, { code: 'CONFLICT', message: `A location with code ${b.code} already exists`, details: [{ field: 'code', issue: 'Location code must be unique' }] });
    const loc = { id: uuid(), code: b.code, name: b.name, description: b.description || '', createdAt: now() };
    locations.set(loc.id, loc);
    return send(res, 201, loc);
  }
  // @endpoint GET /locations/:locationId
  if (method === 'GET' && mLocId) {
    if (!checkApiKey(req, res)) return;
    const loc = locations.get(mLocId[1]);
    if (!loc) return send(res, 404, { code: 'NOT_FOUND', message: `Location ${mLocId[1]} was not found`, details: [] });
    return send(res, 200, loc);
  }

  // ITEMS ───────────────────────────────────────────────────────────────────────
  // @endpoint GET /items
  if (method === 'GET' && path === '/items') {
    if (!checkApiKey(req, res)) return;
    return send(res, 200, paginate([...items.values()], url));
  }
  // @endpoint POST /items
  if (method === 'POST' && path === '/items') {
    if (!checkApiKey(req, res)) return;
    const b = await getBody(req);
    const miss = [];
    if (!b.sku)  miss.push({ field: 'sku',  issue: 'required field is missing' });
    if (!b.name) miss.push({ field: 'name', issue: 'required field is missing' });
    if (!b.unit) miss.push({ field: 'unit', issue: 'required field is missing' });
    if (miss.length) return send(res, 400, { code: 'VALIDATION_ERROR', message: 'Request body is missing required fields', details: miss });
    if ([...items.values()].some(i => i.sku === b.sku))
      return send(res, 409, { code: 'CONFLICT', message: `An item with SKU ${b.sku} already exists`, details: [{ field: 'sku', issue: 'SKU must be unique across all items' }] });
    const item = { id: uuid(), sku: b.sku, name: b.name, description: b.description || '', unit: b.unit,
                   reorderThreshold: b.reorderThreshold ?? 0, status: b.status || 'active', createdAt: now(), updatedAt: now() };
    items.set(item.id, item);
    return send(res, 201, item);
  }
  // GET /items/:itemId/stock  — must precede the generic /items/:itemId route
  // @endpoint GET /items/:itemId/stock
  if (method === 'GET' && mStock) {
    if (!checkApiKey(req, res)) return;
    const itemId = mStock[1];
    const rows = [];
    for (const [key, qty] of stock.entries()) {
      const [kItem, kLoc] = key.split('|');
      if (kItem === itemId) rows.push({ itemId, locationId: kLoc, quantity: qty, updatedAt: now() });
    }
    return send(res, 200, paginate(rows, url));
  }
  // @endpoint GET /items/:itemId
  if (method === 'GET' && mItemId) {
    if (!checkApiKey(req, res)) return;
    const item = items.get(mItemId[1]);
    if (!item) return send(res, 404, { code: 'NOT_FOUND', message: `Item ${mItemId[1]} was not found`, details: [] });
    return send(res, 200, item);
  }
  // @endpoint PATCH /items/:itemId
  if (method === 'PATCH' && mItemId) {
    if (!checkApiKey(req, res)) return;
    const item = items.get(mItemId[1]);
    if (!item) return send(res, 404, { code: 'NOT_FOUND', message: `Item ${mItemId[1]} was not found`, details: [] });
    const b = await getBody(req);
    const updated = { ...item, ...b, id: item.id, updatedAt: now() };
    items.set(item.id, updated);
    return send(res, 200, updated);
  }
  // @endpoint DELETE /items/:itemId
  if (method === 'DELETE' && mItemId) {
    if (!checkApiKey(req, res)) return;
    if (!items.has(mItemId[1])) return send(res, 404, { code: 'NOT_FOUND', message: `Item ${mItemId[1]} was not found`, details: [] });
    items.delete(mItemId[1]);
    res.writeHead(204);
    return res.end();
  }

  // STOCK MOVEMENTS ─────────────────────────────────────────────────────────────
  // @endpoint GET /stock-movements
  if (method === 'GET' && path === '/stock-movements') {
    if (!checkApiKey(req, res)) return;
    const fItem = url.searchParams.get('itemId');
    const fLoc  = url.searchParams.get('locationId');
    const fType = url.searchParams.get('type');
    let list = movements.slice();
    if (fItem) list = list.filter(m => m.itemId     === fItem);
    if (fLoc)  list = list.filter(m => m.locationId === fLoc);
    if (fType) list = list.filter(m => m.type       === fType);
    return send(res, 200, paginate(list, url));
  }
  // @endpoint POST /stock-movements
  if (method === 'POST' && path === '/stock-movements') {
    if (!checkApiKey(req, res)) return;
    const b = await getBody(req);
    const miss = [];
    if (!b.type)            miss.push({ field: 'type',       issue: 'required field is missing' });
    if (!b.itemId)          miss.push({ field: 'itemId',     issue: 'required field is missing' });
    if (!b.locationId)      miss.push({ field: 'locationId', issue: 'required field is missing' });
    if (b.quantity == null) miss.push({ field: 'quantity',   issue: 'required field is missing' });
    if (miss.length) return send(res, 400, { code: 'VALIDATION_ERROR', message: 'Request body is missing required fields', details: miss });
    if (typeof b.quantity !== 'number' || !Number.isInteger(b.quantity) || b.quantity < 1)
      return send(res, 400, { code: 'VALIDATION_ERROR', message: 'quantity must be a positive integer (minimum 1)',
                              details: [{ field: 'quantity', issue: `received ${b.quantity}, must be >= 1` }] });
    const key     = `${b.itemId}|${b.locationId}`;
    const current = stock.get(key) || 0;
    if (b.type === 'issue' && b.quantity > current)
      return send(res, 422, { code: 'INSUFFICIENT_STOCK', message: `Cannot issue ${b.quantity} units; only ${current} available`,
                              details: [{ field: 'quantity', issue: `requested ${b.quantity}, available ${current}` }] });
    if      (b.type === 'receive') stock.set(key, current + b.quantity);
    else if (b.type === 'issue')   stock.set(key, current - b.quantity);
    else if (b.type === 'adjust')  stock.set(key, Math.max(0, current + (b.delta || 0)));
    const mv = { id: uuid(), type: b.type, itemId: b.itemId, locationId: b.locationId,
                 quantity: b.quantity, referenceNumber: b.referenceNumber || null, createdAt: now() };
    movements.push(mv);
    return send(res, 201, mv);
  }

  send(res, 404, { code: 'NOT_FOUND', message: 'Endpoint not defined', details: [] });
});

server.listen(PORT);
