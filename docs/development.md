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
- `documents`
- `processing_jobs`

### 2. Start React frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend auto-registers and logs in a dev user on first load.

## Environment variables

All variables live in `.env` at the project root (see `.env.example`).

### Backend

| Variable | Default | Description |
| --- | --- | --- |
| `OCR_PROVIDER` | `google_vision` | OCR provider name (`google_vision`) |
| `OCR_API_KEY` | empty | Google Cloud Vision API key (required) |
| `OCR_RESULT_LANGUAGE` | empty | ISO 639-1 code (e.g. `en`, `de`). When set, `title`, `summary`, `purpose`, and `document_type` are stored in this language; originals go in `*_original` fields. Tags and document types are created in both languages when a translation is available. |
| `OPENAI_API_KEY` | empty | OpenAI-compatible API key |
| `OPENAI_MODEL` | `gpt-4o-mini` | Model ID for metadata extraction |
| `OPENAI_CHAT_MODEL` | `OPENAI_MODEL` | Optional model ID for document chat |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI-compatible API base URL |
| `OPENAI_TIMEOUT_SEC` | `60` | AI request timeout |
| `WORKER_CRON_EXPR` | `* * * * *` | Cron expression for sweeping stuck pending jobs (minute granularity; new jobs dispatch immediately via hooks) |
| `WORKER_MAX_RETRIES` | `3` | Max AI retry attempts per job |
| `EXTRACTION_PROMPT_VERSION` | `v1` | Stored on each processing job |

### Frontend

| Variable | Default | Description |
| --- | --- | --- |
| `VITE_POCKETBASE_URL` | `http://127.0.0.1:8090` | PocketBase API URL |
| `VITE_DEV_USER_EMAIL` | `dev@paperless.local` | Dev auth email |
| `VITE_DEV_USER_PASSWORD` | `devpassword123` | Dev auth password |

## Processing flow

1. User uploads a document from `/upload`
2. PocketBase stores the file and creates a `processing_jobs` record via Go hook
3. An `OnRecordAfterCreateSuccess` hook dispatches the job immediately; a cron job (`process_pending_jobs`) sweeps any stuck pending jobs
4. Worker generates a PNG preview from the first PDF page (via `pdftoppm`), then runs OCR and AI extraction
5. Extracted metadata is saved on the document
6. UI shows status on list and detail pages

Cron jobs are visible and manually triggerable in PocketBase Admin → Settings → Crons.

## OpenAI setup

1. Create an API key for OpenAI or another OpenAI-compatible provider.
2. Set `OPENAI_API_KEY` in `.env`.
3. Optionally change `OPENAI_MODEL`, `OPENAI_CHAT_MODEL`, or `OPENAI_BASE_URL`.

Without an API key, AI extraction and document chat return a configuration error.

## OCR setup

Uses the official [Go client library](https://docs.cloud.google.com/vision/docs/detect-labels-image-client-libraries).

```env
OCR_PROVIDER=google_vision
OCR_API_KEY=your-google-api-key
```

- **Images** — `BatchAnnotateImages` with `DOCUMENT_TEXT_DETECTION` via `images:annotate`
- **PDFs** — `BatchAnnotateFiles` via `files:annotate` (base64 upload, no Cloud Storage). Pages are processed in batches of up to 5 per request.

## Useful commands

```bash
# Backend tests
cd backend && go test ./...

# Frontend build (outputs to ../public for PocketBase to serve)
cd frontend && npm run build

# Create a new migration
cd backend && go run . migrate create "your_migration_name"
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
- **OCR fails:** verify `OCR_API_KEY` is set and Vision API is enabled for your Google Cloud project.
- **AI extraction fails:** verify `OPENAI_API_KEY`, `OPENAI_BASE_URL`, and model name. Check the processing job error on the document detail page.
- **Auth errors in frontend:** delete PocketBase data dir (`backend/pb_data`) and restart to recreate collections, then reload the app.
