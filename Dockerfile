FROM golang:1.16.2-alpine AS build

WORKDIR /tmp/app

COPY . .

RUN apk add --no-cache git && \
    go mod download && \
    go mod verify && \
    go build -o kitsune-bot

FROM alpine:latest

WORKDIR /home/kitsune-bot

COPY --from=build /tmp/app/kitsune-bot /home/kitsune-bot/

EXPOSE 80

ENTRYPOINT ./kitsune-bot