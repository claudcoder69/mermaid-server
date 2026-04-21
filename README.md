# mermaid-server

An HTTP server that renders [Mermaid](https://mermaid.js.org/) diagrams by
wrapping [`@mermaid-js/mermaid-cli`](https://github.com/mermaid-js/mermaid-cli).

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
