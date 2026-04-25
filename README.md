# mermaid-server

> **This is a maintained fork of [TomWright/mermaid-server](https://github.com/TomWright/mermaid-server).** ❤️
>
> Enormous thanks to [Tom Wright](https://github.com/TomWright) for building
> the original project and for keeping it running for years. ❤️ All of the
> original design — the simple `/generate` endpoint, the in-memory cache, the
> `mmdc` wrapper — is his work. This fork just dusts it off and picks up active
> maintenance. ❤️

An HTTP server that renders [Mermaid](https://mermaid.js.org/) diagrams by
wrapping [`@mermaid-js/mermaid-cli`](https://github.com/mermaid-js/mermaid-cli).

## What changed in this fork

**The REST API is unchanged** — existing clients against `GET|POST /generate`
continue to work with no modifications.

Under the hood:

- **Toolchain refresh** — Go `1.15` → `1.23`, Debian `buster` → `bookworm`,
  Node `18` → `22`, `@mermaid-js/mermaid-cli` `9.x` → `11.x`.
- **Unmaintained dependencies removed** — the old `tomwright/grace` and
  `tomwright/gracehttpserverrunner` packages are replaced by stdlib
  `signal.NotifyContext` + `http.Server.Shutdown`.
- **Router swap** — the stdlib `http.ServeMux` is replaced with
  [`go-chi/chi`](https://github.com/go-chi/chi). Routes, methods, and paths are
  identical.
- **OpenAPI 3.0 spec** served at `GET /openapi.yaml`, with a Swagger UI at
  `GET /docs`.
- **New `GET /health`** endpoint for container healthchecks.
- **Default listen port** is now `:8080` (was `:80`).
- **HTTP server timeouts** (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout`) are configured.
- **1 MiB POST body cap**; `mmdc` invocations are wrapped in a 60s timeout.
- **Bug fixes** — fixed an un-flushed `bufio.Writer` that silently dropped
  `mmdc` stdout/stderr from logs, fixed `.png` output files being orphaned by
  cleanup, and added the missing mutex on the in-memory cache.
- **Unit tests** added (pass under `-race`).
- **CI** rewired to use `actions/checkout@v4`, `actions/setup-go@v5`, and
  `docker/build-push-action@v6` with GitHub Actions layer cache. The
  Docker-build workflow now runs on every PR and smoke-tests `/health`; it no
  longer publishes images.

## Running

### Docker

```
docker run -d --name mermaid-server -p 8080:8080 tomwright/mermaid-server:latest
```

### Locally

```
cd mermaidcli && npm install && cd ..
go run ./cmd/app \
    --mermaid=./mermaidcli/node_modules/.bin/mmdc \
    --in=./in \
    --out=./out
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--mermaid` | (required) | Path to the `mmdc` executable. |
| `--in` | (required) | Directory used to stage input `.mmd` files. |
| `--out` | (required) | Directory used for generated images. |
| `--addr` | `:8080` | Address the HTTP server listens on. |
| `--puppeteer` | `""` | Optional path to a puppeteer config file. |
| `--allow-all-origins` | `false` | Enable permissive CORS (`Access-Control-Allow-Origin: *`) and preflight handling. |

## API

Interactive documentation is available at
[`/docs`](http://localhost:8080/docs), backed by the raw OpenAPI 3.0 spec at
[`/openapi.yaml`](http://localhost:8080/openapi.yaml).

### `GET /health`

Returns `200 OK` with body `ok`. Used for container health checks.

### `GET|POST /generate`

Generates a diagram. The `type` query parameter controls the output format and
may be `svg` (default) or `png`.

#### POST

Send the Mermaid source as the request body:

```
curl --location --request POST 'http://localhost:8080/generate' \
    --header 'Content-Type: text/plain' \
    --data-raw 'graph LR
    A-->B
    B-->C
    C-->D
    C-->F
'
```

Request bodies are limited to 1 MiB.

#### GET

Send URL-encoded Mermaid source under the `data` query parameter:

```
curl --location --request GET 'http://localhost:8080/generate?data=graph%20LR%0A%0A%20%20%20%20A--%3EB%0A%20%20%20%20B--%3EC%0A%20%20%20%20C--%3ED%0A%20%20%20%20C--%3EF%0A'
```

![Example request in Postman](example.png "Example request in Postman")

## Caching

Generated diagrams are cached in memory and on disk for one hour after their
last access. The cleanup loop runs every five minutes.
