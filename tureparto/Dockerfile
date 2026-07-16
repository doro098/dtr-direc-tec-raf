FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o tureparto-server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o cliente-tureparto ./cmd/cliente

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/tureparto-server .
COPY --from=builder /app/cliente-tureparto .
COPY --from=builder /app/internal/visor/static ./internal/visor/static
EXPOSE 3000 3001
CMD ["./tureparto-server"]
