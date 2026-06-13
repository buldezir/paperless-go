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
| `OPENCODE_GO_API_KEY` | empty | OpenCode Go API key |
| `OPENCODE_GO_MODEL` | `deepseek-v4-flash` | OpenCode Go model ID |
| `OPENCODE_GO_BASE_URL` | `https://opencode.ai/zen/go/v1` | OpenCode Go API base URL |
| `OPENCODE_GO_TIMEOUT_SEC` | `60` | AI request timeout |
| `WORKER_POLL_INTERVAL_SEC` | `5` | Background worker poll interval |
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
3. Background worker picks pending jobs
4. Worker generates a PNG preview from the first PDF page (via `pdftoppm`), then runs OCR and AI extraction
5. Extracted metadata is saved on the document
6. UI shows status on list and detail pages

## OpenCode Go setup

1. Subscribe and create an API key at [OpenCode Go](https://opencode.ai/docs/go/)
2. Set `OPENCODE_GO_API_KEY` in `.env`
3. Optionally change `OPENCODE_GO_MODEL` (for example `deepseek-v4-flash` or `glm-5`)

Without an API key, the backend uses a mock AI extractor for local development.

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

## Troubleshooting

- **Upload succeeds but stays pending:** ensure the backend server is running; the worker starts with `serve`.
- **OCR fails:** verify `OCR_API_KEY` is set and Vision API is enabled for your Google Cloud project.
- **AI extraction fails:** verify `OPENCODE_GO_API_KEY` and model name. Check the processing job error on the document detail page.
- **Auth errors in frontend:** delete PocketBase data dir (`backend/pb_data`) and restart to recreate collections, then reload the app.
