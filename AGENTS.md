# Agent instructions

## Verification (required)

Before considering a task done, run the full verification stack and fix failures:

```bash
./scripts/test-all.sh
```

That covers:

1. Backend unit tests (`go test`, excluding `/e2e`)
2. Backend API e2e (`go test ./e2e/`)
3. Frontend Playwright e2e (`npm run test:e2e`)

Do not claim the task is complete if any stage fails. Prefer the full script over running only the package you touched.

A task is incomplete until `./scripts/test-all.sh` passes and related tests reflect the new behavior.

## Tests must stay in sync

When changing existing behavior:

- Update or add unit tests for the affected packages.
- Update API e2e under `backend/e2e/` when HTTP/API behavior changes.
- Update Playwright specs under `frontend/` when UI flows change.
- Do not leave tests asserting the old behavior; change production code and tests together.
- Prefer extending existing tests over skipping or deleting coverage.

New features need tests at the same layer as similar code already has.
