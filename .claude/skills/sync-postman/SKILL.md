---
name: sync-postman
description: Pull or push Postman collections/environments between this repo's postman/ folders and the linked cloud workspace, via the Postman CLI. User-triggered only (/sync-postman pull|push) — never invoke this automatically.
disable-model-invocation: true
---

Direction is given as `$ARGUMENTS`: `pull` (cloud workspace → local `postman/` folders) or `push` (local → cloud). If `$ARGUMENTS` is empty or anything other than `pull`/`push`, ask the user which direction they mean before doing anything.

Workspace id (from `.postman/resources.yaml`): `c1e6f8e5-8f97-45c3-9b85-67d3cd809f1f`.

Steps:

1. Check the Postman CLI is available: `which postman`. If missing, stop and tell the user to install it (`npm install -g postman-cli` or see https://learning.postman.com/docs/postman-cli/postman-cli-installation/) and run `postman login` first — do not attempt a workaround.
2. Check auth: `postman --version` succeeding doesn't imply login — if a command fails with an auth error, tell the user to run `postman login`.
3. For `pull`: fetch collections/environments from workspace `c1e6f8e5-8f97-45c3-9b85-67d3cd809f1f` and write them into the matching empty folders (`postman/collections/`, `postman/environments/`, `postman/mocks/`), preserving Postman's expected file naming/format for Spec Hub local sync.
4. For `push`: push local changes under those folders back to the same workspace.
5. Report what changed (files added/updated) — don't silently overwrite without summarizing the diff first if files already exist locally.

This repo currently has empty `postman/collections/`, `environments/`, `mocks/` folders — a `pull` is expected to populate them for the first time from the cloud-generated collection/E2E suite described in `spec_design/agentmode_e2e_test_prompt.txt`.
