FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o paperless-go .

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/paperless-go /app/paperless-go
EXPOSE 8090
CMD ["/app/paperless-go", "serve", "--http=0.0.0.0:8090"]
