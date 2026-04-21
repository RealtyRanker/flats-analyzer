FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /flats-analyzer ./cmd/analyzer

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /var/log/flats-analyzer

WORKDIR /app
COPY --from=builder /flats-analyzer .
COPY config.yaml .

EXPOSE 9093

CMD ["./flats-analyzer", "-config", "config.yaml"]
