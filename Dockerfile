# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /relay ./cmd/relay

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /relay /usr/local/bin/relay

EXPOSE 8080
ENTRYPOINT ["relay"]
CMD ["server"]
