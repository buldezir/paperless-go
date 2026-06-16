# Paperless Go

Document storage MVP built with Go + PocketBase and a React + TanStack Router frontend. Upload documents, run OCR, extract metadata with an OpenAI-compatible AI provider, and review results in the UI.

## Paperless-ngx compatibility

Paperless Go implements a [paperless-ngx](https://github.com/paperless-ngx/paperless-ngx)-compatible REST API under `/api/`, so you can use third-party clients instead of (or alongside) the built-in web UI. Coverage is partial — document list/upload/download, tags, and metadata generally work, but not every paperless-ngx endpoint or feature is implemented.

The API has been tested with the [swift-paperless](https://github.com/paulgessinger/swift-paperless) iOS app and mostly works for browsing and uploading documents. See [docs/development.md](docs/development.md#paperless-ngx-api-compatibility) for connecting external clients.

## Stack

- **Backend:** Go, [PocketBase as a framework](https://pocketbase.io/docs/use-as-framework/)
- **Frontend:** React, TanStack Router, PocketBase JS SDK
- **OCR:** Google Cloud Vision (`google_vision`) or Mistral (`mistral`), selected via `OCR_PROVIDER`
- **AI:** OpenAI-compatible chat completions via the official OpenAI Go SDK

## Project layout

```text
backend/    PocketBase app, migrations, OCR/AI worker
frontend/   React UI
docs/       Development guide
```

## Quick start

```bash
cp .env.example .env
# Edit .env and set OCR provider keys and OPENAI_API_KEY
docker compose up --build
```

Open [http://127.0.0.1:8090](http://127.0.0.1:8090). Data is stored in a Docker volume (`app_data`).

For local development without Docker, see [docs/development.md](docs/development.md).

## Environment variables

See [docs/development.md](docs/development.md) for the full list.

Minimum for local dev:

- Set `GOOGLE_VISION_API_KEY` (for `OCR_PROVIDER=google_vision`) or `MISTRAL_API_KEY` (for `OCR_PROVIDER=mistral`)
- Set `OPENAI_API_KEY` to use AI extraction

## Features

- Upload PDF, image, or plain text documents
- Async processing jobs with status tracking
- OCR text extraction
- AI metadata extraction: title, purpose, date, type, tags, summary
- Document list with search and status filters
- Detail page for reviewing OCR text and correcting metadata

## Tests

```bash
cd backend && go test ./...
cd frontend && npm run build
```
