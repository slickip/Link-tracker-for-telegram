FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /scrapper ./scrapper/cmd

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /scrapper /scrapper

EXPOSE 8081 8083

CMD ["/scrapper"]