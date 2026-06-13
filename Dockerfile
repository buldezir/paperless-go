FROM golang:1.24-alpine AS backend-builder

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o paperless-go .

FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

FROM alpine:3.21
RUN apk add --no-cache poppler-utils
WORKDIR /app
COPY --from=backend-builder /app/backend/paperless-go /app/paperless-go
COPY --from=frontend-builder /app/public /app/public
EXPOSE 8090
CMD ["/app/paperless-go", "serve", "--http=0.0.0.0:8090"]
