# Development Guide

## Prerequisites

- Go 1.23+
- Node.js 20+
- npm
- [poppler-utils](https://poppler.freedesktop.org/) (`pdftoppm`) for PDF preview thumbnails

On macOS: `brew install poppler`. On Debian/Ubuntu: `apt install poppler-utils`.

## Running locally

```bash
cp .env.example .env
```

### 1. Start PocketBase backend

```bash
cd backend
go run . serve --http=127.0.0.1:8090
```

On first run, migrations create:

- `tags`
- `correspondents`
- `document_types`
- `documents`
- `processing_jobs`
- `app_settings` (singleton; seeded from `.env` on first boot)

### 2. Start React frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend auto-logs in a regular `users` account when `VITE_DEV_*` is set.

## Environment variables

All variables live in `.env` at the project root (see `.env.example`).

### Always env-backed

| Variable | Default | Description |
| --- | --- | --- |
| `WORKER_CRON_EXPR` | `* * * * *` | Cron expression for sweeping stuck pending jobs (registered once at startup) |
| `VITE_POCKETBASE_URL` | `http://127.0.0.1:8090` | PocketBase API URL (frontend) |
| `VITE_DEV_USER_EMAIL` | — | Dev auto-login email (`users` collection) |
| `VITE_DEV_USER_PASSWORD` | — | Dev auto-login password |

### Seed-only (first boot → Settings)

These seed `app_settings` when the singleton record does not exist yet. After that, edit them in the app **Settings** page (requires a PocketBase superuser login). Changing `.env` alone will not update a running install.

| Variable | Default | Description |
| --- | --- | --- |
| `OCR_PROVIDER` | `google_vision` | OCR provider (`google_vision`, `mistral`) |
| `GOOGLE_VISION_API_KEY` | empty | Google Cloud Vision API key |
| `MISTRAL_API_KEY` | empty | Mistral API key |
| `MISTRAL_OCR_MODEL` | `mistral-ocr-latest` | Mistral OCR model |
| `MISTRAL_API_BASE_URL` | `https://api.mistral.ai/v1` | Mistral API base URL |
| `OCR_TIMEOUT_SEC` | `40` | OCR request timeout |
| `PROCESSING_RESULT_LANGUAGE` | empty | ISO 639-1 code (e.g. `en`, `de`). When set, `title`, `summary`, `purpose`, and `document_type` are stored in this language; originals go in `*_original` fields. |
| `OPENAI_API_KEY` | empty | OpenAI-compatible API key |
| `OPENAI_MODEL` | `gpt-4o-mini` | Model ID for metadata extraction |
| `OPENAI_CHAT_MODEL` | `OPENAI_MODEL` | Optional model ID for document chat |
| `OPENAI_SEARCH_MODEL` | `OPENAI_CHAT_MODEL` | Optional model ID for Deep Search |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI-compatible API base URL |
| `OPENAI_TIMEOUT_SEC` | `60` | AI request timeout |
| `DEEP_SEARCH_LANGUAGES` | empty | Comma-separated ISO 639-1 codes (e.g. `de,en,uk`) for Deep Search keyword expansion |
| `WORKER_TIMEOUT_SEC` | `300` | Per-job processing timeout |
| `WORKER_MAX_RETRIES` | `0` | Max step retry attempts before a job fails |
| `EXTRACTION_PROMPT_VERSION` | `v1` | Stored on each processing job step run |

## Settings (admin UI)

1. Create a PocketBase superuser if you do not have one yet (installer UI on first run, or `go run . superuser upsert EMAIL PASS` from `backend/`).
2. Sign out of the app if you are logged in as a regular user, then sign in with the **superuser** email/password (login tries `users`, then `_superusers`).
3. Open **Settings** in the nav. Changes save to `app_settings` and hot-reload the in-process OCR/AI clients (no restart).

`WORKER_CRON_EXPR` is not editable there; change `.env` and restart, or use PocketBase Admin → Settings → Crons.

## Processing flow

1. User uploads a document from `/upload`
2. PocketBase stores the file and creates a `processing_jobs` record via Go hook
3. An `OnRecordAfterCreateSuccess` hook dispatches the job immediately; a cron job (`process_pending_jobs`) sweeps any stuck pending jobs
4. Worker generates a PNG preview from the first PDF page (via `pdftoppm`), then runs OCR and AI extraction
5. Extracted metadata is saved on the document
6. UI shows status on list and detail pages

Cron jobs are visible and manually triggerable in PocketBase Admin → Settings → Crons.

## OpenAI setup

Prefer **Settings** in the UI (superuser). For a fresh install you can also put `OPENAI_API_KEY` (and optional model/base URL) in `.env` so they seed `app_settings` on first boot.

Without an API key, AI extraction, document chat, and Deep Search return a configuration error.

Deep Search (`/search`) uses a tool-calling agent over keyword search. Configure **Search model** and **Deep search languages** in Settings.

## OCR setup

Set the provider and API keys in **Settings** (or seed via `.env` on first boot).

### Google Cloud Vision

Uses the official [Go client library](https://docs.cloud.google.com/vision/docs/detect-labels-image-client-libraries).

- **Images** — `BatchAnnotateImages` with `DOCUMENT_TEXT_DETECTION` via `images:annotate`
- **PDFs** — `BatchAnnotateFiles` via `files:annotate` (base64 upload, no Cloud Storage). Pages are processed in batches of up to 5 per request.

See [docs/google_vision.md](google_vision.md) for obtaining a Google API key.

### Mistral OCR

Uses the [Mistral Document OCR API](https://docs.mistral.ai/en/studio-api/document-processing/basic_ocr). Local files are sent as base64 data URLs (up to 10 MB).

- **PDFs and office documents** — `document_url` with a base64 data URL
- **Images** — `image_url` with a base64 data URL
- **Output** — page markdown joined into plain text

## Useful commands

```bash
# Backend tests
cd backend && go test ./...

# Frontend build (outputs to ../public for PocketBase to serve)
cd frontend && npm run build

# Create a new migration
cd backend && go run . migrate create "your_migration_name"

# Create / update a PocketBase superuser
cd backend && go run . superuser upsert admin@example.com 'your-password'
```

## Paperless-ngx API compatibility

Paperless Go exposes a paperless-ngx-compatible REST API on the same host as PocketBase (for example `http://127.0.0.1:8090/api/`). The backend implements the endpoints third-party clients expect for authentication, documents, tags, correspondents, document types, and related metadata.

Compatibility is intentionally partial: common read/write flows work, but not every paperless-ngx feature is available (for example, some list endpoints return empty stubs where the MVP has no equivalent data).

### Connecting external clients

1. Point the client at your Paperless Go server URL (scheme + host + port, no `/api` suffix — clients add that themselves).
2. Sign in with a PocketBase user account. The `/api/token/` endpoint accepts the same username and password as the web UI.
3. Clients that send `Authorization: Token <jwt>` (paperless-ngx style) are supported alongside standard Bearer tokens.

API versions 9 and 10 are accepted via the `Accept` header (`application/json; version=9`).

### swift-paperless (iOS)

[swift-paperless](https://github.com/paulgessinger/swift-paperless) is the main mobile client exercised against this API. Browsing documents, viewing details, and uploading generally work. Some paperless-ngx-specific settings or advanced features may be missing or no-ops because Paperless Go does not implement the full paperless-ngx surface area.

## Troubleshooting

- **Upload succeeds but stays pending:** ensure the backend server is running; the worker starts with `serve`.
- **OCR fails:** configure the OCR provider and API key in Settings (or seed `.env` before first boot). For Google Vision, ensure the Vision API is enabled for your project.
- **AI extraction fails:** configure OpenAI settings in Settings. Check the processing job error on the document detail page.
- **Settings page missing:** log in with a PocketBase `_superusers` account (not a regular `users` account).
- **Auth errors in frontend:** delete PocketBase data dir (`backend/pb_data`) and restart to recreate collections, then reload the app.
