# Paperless Go

Document storage MVP built with Go + PocketBase and a React + TanStack Router frontend. Upload documents, run OCR, extract metadata with OpenCode Go AI, and review results in the UI.

## Stack

- **Backend:** Go, [PocketBase as a framework](https://pocketbase.io/docs/use-as-framework/)
- **Frontend:** React, TanStack Router, PocketBase JS SDK
- **OCR:** Google Cloud Vision (requires `OCR_API_KEY`)
- **AI:** [OpenCode Go](https://opencode.ai/docs/go/) with mock fallback when no API key is set

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
- Set `OPENCODE_GO_API_KEY` to use AI extraction

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
