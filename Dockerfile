# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine AS build-server

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

FROM golang:1.27.3-alpine AS build-hook-dispatch

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal

ENV CGO_ENABLED=0
RUN go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/hayel-hook-dispatch \
    ./cmd/hayel-hook-dispatch

FROM scratch AS hook-dispatch

COPY --from=build-hook-dispatch /out/hayel-hook-dispatch /usr/local/bin/hayel-hook-dispatch

FROM alpine:3.24

ENV HAYEL_TEMPLATE_DIRECTORY=/opt/hayel/template
ENV HAYEL_HOOK_ON=1

RUN apk add --no-cache ca-certificates git git-daemon \
    && addgroup -S -g 1000 hayel \
    && adduser -S -G hayel -u 1000 hayel \
    && mkdir -p /repositories /opt/hayel/template/hooks \
    && ln -sf /usr/local/bin/hayel-hook-dispatch /opt/hayel/template/hooks/pre-receive \
    && ln -sf /usr/local/bin/hayel-hook-dispatch /opt/hayel/template/hooks/update \
    && ln -sf /usr/local/bin/hayel-hook-dispatch /opt/hayel/template/hooks/post-receive \
    && ln -sf /usr/local/bin/hayel-hook-dispatch /opt/hayel/template/hooks/post-update \
    && chown -R 1000:1000 /repositories /opt/hayel/template

COPY --from=build-server /out/hayel-server /usr/local/bin/hayel-server
COPY --from=hook-dispatch /usr/local/bin/hayel-hook-dispatch /usr/local/bin/hayel-hook-dispatch

USER 1000:1000
VOLUME ["/repositories"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/hayel-server"]
