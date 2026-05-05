FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./bot/cmd

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bot /bot

EXPOSE 8080 8082

CMD ["/bot"]