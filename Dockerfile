#1 этап сборка

FROM golang:1.26.8-alpine AS builder


WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -o /app/main ./cmd/api

#2 этап запуск

FROM alpine:latest

WORKDIR /app

#просим из нашего этапа builder взять только бинарник, оставляя весь "мусор", в виде компилятор
#и прочего на прошлом этапе, они на не нужны, нужен только бинарник
COPY --from=builder /app/main .

RUN chmod +x ./main #выдача прав

EXPOSE 8080

CMD ["./main"]



