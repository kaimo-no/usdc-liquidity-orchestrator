# Multi-stage build for the plan-only HTTP microservice (cmd/server).
# No secrets required. Inventory is request-scoped only.

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
