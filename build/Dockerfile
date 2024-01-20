FROM golang:1.21-alpine3.19 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o gophermart ./cmd/gophermart/*.go

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/gophermart gophermart

ENTRYPOINT ["/app/gophermart"]