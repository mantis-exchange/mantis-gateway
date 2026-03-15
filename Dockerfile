# ---------- builder ----------
FROM golang:1.23-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /mantis-gateway ./cmd/gateway

# ---------- runtime ----------
FROM gcr.io/distroless/static-debian12

COPY --from=builder /mantis-gateway /mantis-gateway

EXPOSE 8080

CMD ["/mantis-gateway"]
