# Build stage

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY . .

RUN go work edit -dropuse ./tests/integration
RUN go work edit -dropuse ./tests/security
RUN go work edit -dropuse ./tests/e2e

WORKDIR /src/pkg/platform

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /httpprobe ./cmd/httpprobe

WORKDIR /src/services/auth

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth ./cmd/auth

# Runtime stage

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /auth /auth
COPY --from=builder /httpprobe /httpprobe

EXPOSE 8080

USER nonroot:nonroot

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 CMD ["/httpprobe", "http://127.0.0.1:8080/api/v1/health"]

ENTRYPOINT ["/auth"]
