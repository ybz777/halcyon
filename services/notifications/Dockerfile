# Build stage

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY . .

RUN go work edit -dropuse ./tests/integration
RUN go work edit -dropuse ./tests/security
RUN go work edit -dropuse ./tests/e2e

WORKDIR /src/pkg/platform

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /httpprobe ./cmd/httpprobe

WORKDIR /src/services/notifications

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /notifications ./cmd/notifications
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /notification-dispatcher ./cmd/dispatcher
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /email-worker ./cmd/worker

# Runtime stage

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /notifications /notifications
COPY --from=builder /notification-dispatcher /notification-dispatcher
COPY --from=builder /email-worker /email-worker
COPY --from=builder /httpprobe /httpprobe

EXPOSE 8085

USER nonroot:nonroot

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 CMD ["/httpprobe", "http://127.0.0.1:8085/api/v1/health"]

ENTRYPOINT ["/notifications"]