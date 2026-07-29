# Multi-stage build for the HTTP microservice (cmd/server):
#   GET /                  plan / scenario / consolidate UI
#   POST /v1/plan          shortfall-only dry plan
#   POST /v1/payment-funding  scenario full-funding dry plan
#   POST /v1/consolidate   gateway deposit plan
#   GET  /v1/chains        corridor registry
#   GET  /healthz
#
# Default image is plan-only (no secrets). Live testnet execute is dual-gated
# and requires loopback LISTEN_ADDR — not suitable for this distroless publish.

FROM golang:1.26.5-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV LISTEN_ADDR=:8088
EXPOSE 8088
USER nonroot:nonroot
ENTRYPOINT ["/server"]
