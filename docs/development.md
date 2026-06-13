# Development Guide

## Prerequisites

- Go 1.23+
- Node.js 20+
- npm

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
| `OCR_PROVIDER` | `mock` | OCR provider name (`mock`, `google_vision`) |
| `OCR_API_KEY` | empty | Google Cloud Vision API key when using `google_vision` |
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
4. Worker runs OCR, then AI extraction
5. Extracted metadata is saved on the document
6. UI shows status on list and detail pages

## OpenCode Go setup

1. Subscribe and create an API key at [OpenCode Go](https://opencode.ai/docs/go/)
2. Set `OPENCODE_GO_API_KEY` in `.env`
3. Optionally change `OPENCODE_GO_MODEL` (for example `deepseek-v4-flash` or `glm-5`)

Without an API key, the backend uses a mock AI extractor for local development.

## OCR setup

### Mock OCR (default)

Works out of the box. Plain text files return their actual content. Other files get sample OCR text suitable for testing AI extraction.

### Google Cloud Vision

```env
OCR_PROVIDER=google_vision
OCR_API_KEY=your-google-api-key
```

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
- **AI extraction fails:** verify `OPENCODE_GO_API_KEY` and model name. Check the processing job error on the document detail page.
- **Auth errors in frontend:** delete PocketBase data dir (`backend/pb_data`) and restart to recreate collections, then reload the app.
