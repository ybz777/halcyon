# Build stage

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY . .

RUN go work edit -dropuse ./tests/integration
RUN go work edit -dropuse ./tests/security
RUN go work edit -dropuse ./tests/e2e

WORKDIR /src/cmd/outbox-relay

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /outbox-relay .

# Runtime stage

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /outbox-relay /outbox-relay

USER nonroot:nonroot

ENTRYPOINT ["/outbox-relay"]