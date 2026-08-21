# Hayel

Running your own single-user, serverless Git server focused on HTTPS and OIDC authentication.

## Commands

- `hayel-server` authenticates requests with OIDC and invokes Git's built-in `git-http-backend` for local repositories.

## Running the server

Create or mount a directory containing bare repositories, then run:

```text
go run ./cmd/hayel-server
```

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

The current boilerplate stores sessions in memory, so restarting the server logs users out. Production hardening should add TLS, a durable session store, and CSRF protection for state-changing operations. Git's `git-http-backend` is executed locally for each request.

Any user successfully authenticated by the configured OIDC provider is authorized to use the server. Hayel does not maintain a separate local user allowlist.

Repositories are addressed by paths such as `/group/repo-name`. A bare repository is created automatically when the first push is received at that path. An authenticated `DELETE /group/repo-name` removes the repository and returns `204 No Content`.

An authenticated `GET /` returns JSON containing all repositories, for example `{"repositories":["group/repo-name"]}`.

## Configuring a local client

You need to configure your IdP that supports OIDC to add Hayel, and have the Client ID to hand.

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
