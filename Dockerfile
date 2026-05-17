FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wallet-api ./cmd/server

FROM alpine:3.19

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/wallet-api .
COPY config.env .

EXPOSE 8080
ENV PORT=8080

CMD ["./wallet-api"]
