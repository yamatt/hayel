# syntax=docker/dockerfile:1

FROM golang:1.26.6-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal

ENV CGO_ENABLED=0
RUN go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/hayel-server \
    ./cmd/hayel-server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates git \
    && addgroup -S hayel \
    && adduser -S -G hayel hayel \
    && mkdir /repositories \
    && chown hayel:hayel /repositories

COPY --from=build /out/hayel-server /usr/local/bin/hayel-server

USER hayel
VOLUME ["/repositories"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/hayel-server"]
