---
name: validate-spec
description: Lints postman/specs/inventory-tracker.yaml against the OpenAPI ruleset with Spectral. Use after editing the spec, or whenever the user asks to validate/check/lint the OpenAPI spec.
---

Run Spectral's built-in OpenAPI ruleset against the spec:

```
npx -y @stoplight/spectral-cli lint postman/specs/inventory-tracker.yaml --ruleset spectral:oas
```

Spectral isn't installed globally in this environment — `npx -y` fetches it on demand (first run downloads the package; later runs are cached).

Report errors and warnings back concisely, grouped by path/line. If there are zero issues, just say so in one line — don't dump the full passing output. If Spectral itself fails to run (e.g. no network, npx unavailable), say so plainly rather than treating it as a spec error.
