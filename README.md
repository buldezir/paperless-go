# Paperless Go

Document storage MVP built with Go + PocketBase and a React + TanStack Router frontend. Upload documents, run OCR, extract metadata with an OpenAI-compatible AI provider, and review results in the UI.

## Paperless-ngx compatibility

Paperless Go implements a [paperless-ngx](https://github.com/paperless-ngx/paperless-ngx)-compatible REST API under `/api/`, so you can use third-party clients instead of (or alongside) the built-in web UI. Coverage is partial — document list/upload/download, tags, and metadata generally work, but not every paperless-ngx endpoint or feature is implemented.

The API has been tested with the [swift-paperless](https://github.com/paulgessinger/swift-paperless) iOS app and mostly works for browsing and uploading documents. See [docs/development.md](docs/development.md#paperless-ngx-api-compatibility) for connecting external clients.

## Stack

- **Backend:** Go, [PocketBase as a framework](https://pocketbase.io/docs/use-as-framework/)
- **Frontend:** React, TanStack Router, PocketBase JS SDK
- **OCR:** Google Cloud Vision (requires `OCR_API_KEY`)
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
```

### Backend

```bash
cd backend
go run . serve --http=127.0.0.1:8090
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Open `http://127.0.0.1:5173` for local dev, or build the frontend and serve it from PocketBase:

```bash
cd frontend && npm run build
cd ../backend && go run . serve --http=127.0.0.1:8090
```

Then open `http://127.0.0.1:8090`.

## Environment variables

See [docs/development.md](docs/development.md) for the full list.

Minimum for local dev:

- Set `OCR_API_KEY` for Google Cloud Vision OCR
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
