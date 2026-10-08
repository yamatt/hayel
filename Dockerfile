# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine AS build

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

RUN apk add --no-cache ca-certificates git git-daemon \
    && addgroup -S -g 1000 hayel \
    && adduser -S -G hayel -u 1000 hayel \
    && mkdir /repositories \
    && chown -R 1000:1000 /repositories

COPY --from=build /out/hayel-server /usr/local/bin/hayel-server

USER 1000:1000
VOLUME ["/repositories"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/hayel-server"]
