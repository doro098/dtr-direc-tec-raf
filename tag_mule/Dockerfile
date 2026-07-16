FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o tagmule ./cmd/tagmule

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/tagmule .
COPY config.yaml .
EXPOSE 8080
CMD ["./tagmule"]
