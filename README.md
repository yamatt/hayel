# Hayel 🐪

[![Test](https://github.com/yamatt/hayel/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/yamatt/hayel/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/yamatt/hayel/graph/badge.svg)](https://codecov.io/gh/yamatt/hayel)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yamatt/hayel)](https://github.com/yamatt/hayel/blob/main/go.mod)
[![License](https://img.shields.io/github/license/yamatt/hayel)](https://github.com/yamatt/hayel/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/yamatt/hayel)](https://github.com/yamatt/hayel/releases/latest)
[![Docker Image](https://img.shields.io/badge/Docker-ghcr.io%2Fyamatt%2Fhayel-blue?logo=docker)](https://ghcr.io/yamatt/hayel)

Running your own single-user, serverless Git server focused on HTTPS and OIDC authentication.

It authenticates requests with OIDC and invokes Git's built-in `git-http-backend` to manage on-disk repositories over HTTP.

Any user successfully authenticated by the configured OIDC provider is authorized to use the server. Hayel does not maintain a separate local user allowlist.

Repositories are addressed by paths such as `/group/repo-name`. A bare repository is created automatically when the first push is received at that path. An authenticated `DELETE /group/repo-name` removes the repository and returns `204 No Content`.

An authenticated `GET /` returns JSON containing all repositories, for example `{"repositories":["group/repo-name"]}`.

The current configuration stores sessions in memory, so if the server restarts, it will have to re-verify the user, which can be slow.

## Running the server

Create or mount a directory containing bare repositories, then run:

```text
go run ./cmd/hayel-server
```

### Configuration

The server accepts configuration from a TOML file, `HAYEL_*` environment variables, or command-line flags. Sources are applied in this order, so later sources override earlier ones: defaults, TOML file, environment variables, then flags. Set `HAYEL_CONFIG` or pass `--config` to select a TOML file.

Example `hayel.toml`:

```toml
listen = ":8080"
repository_root = "/repositories"
oidc_issuer = "https://issuer.example.com"
oidc_client_id = "hayel"
oidc_client_secret = "change-me"
oidc_redirect_url = "https://git.example.com/auth/callback"
```

The corresponding environment variables and command-line flags are:

| Variable                                            | Default         | Description                                                                  |
| --------------------------------------------------- | --------------- | ---------------------------------------------------------------------------- |
| `HAYEL_LISTEN` / `--listen`                         | `:8080`         | Address for the Hayel gateway                                                |
| `HAYEL_REPOSITORY_ROOT` / `--repository-root`       | `/repositories` | Directory containing bare Git repositories                                   |
| `HAYEL_OIDC_ISSUER` / `--oidc-issuer`               | —               | OIDC issuer URL                                                              |
| `HAYEL_OIDC_CLIENT_ID` / `--oidc-client-id`         | —               | OIDC client ID                                                               |
| `HAYEL_OIDC_CLIENT_SECRET` / `--oidc-client-secret` | —               | OIDC client secret                                                           |
| `HAYEL_OIDC_REDIRECT_URL` / `--oidc-redirect-url`   | —               | Registered callback URL, for example `https://git.example.com/auth/callback` |

## Configuring your local Git

You need to add an OIDC app to your IdP. The app will need Public Clients and PKCE.

You will need the Client ID to configure Git.

Then:

1. [You need git-credentials-oauth installed](https://github.com/hickford/git-credential-oauth).
1. Configure as follows:

```bash
git config --global credential.https://git-server.example.com.oauthClientId "CLIENT_ID"
git config --global credential.https://git-server.example.com.oauthScopes "openid profile email groups"

git config --global credential.https://git-server.example.com.oauthAuthURL "https://idp.exmaple.com/authorize"
git config --global credential.https://git-server.example.com.oauthTokenURL "https://idp.exmaple.com/api/oidc/token"
```

You should also configure Git to cache the tokens from your IdP

```bash
git config --global credential.https://git-server.example.com.helper "cache --timeout 21600"
git config --global --add credential.https://git-server.example.com.helper oauth
```
