# Build stage

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY . .

RUN go work edit -dropuse ./tests/integration
RUN go work edit -dropuse ./tests/security
RUN go work edit -dropuse ./tests/e2e

WORKDIR /src/pkg/platform

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /httpprobe ./cmd/httpprobe

WORKDIR /src/services/jobs

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /jobs ./cmd/jobs
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /job-worker ./cmd/worker

# Runtime stage

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /jobs /jobs
COPY --from=builder /job-worker /job-worker
COPY --from=builder /httpprobe /httpprobe

EXPOSE 8086

USER nonroot:nonroot

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 CMD ["/httpprobe", "http://127.0.0.1:8086/api/v1/health"]

ENTRYPOINT ["/jobs"]