# go-reader

simple and fast html -> markdown reader service in go.

## routes

- `GET /healthz`
- `GET /readyz`
- `GET /read?url=https://example.com`
- `POST /extract`

## getting started

```bash
docker compose up --build
```

this starts:

- `go-reader` on `:8080`

## optional api key auth

set `API_KEY` to require an `X-API-KEY` header on `GET /read` and `POST /extract`.

- when `API_KEY` is not set, auth is disabled
- `GET /healthz` and `GET /readyz` are always publicx

example:

```bash
API_KEY=supersecret docker compose up --build
```

## extract request

```json
{
  "url": "https://example.com",
  "timeout_ms": 8000,
  "user_agent": "my-reader/1.0"
}
```

for `/extract`, user-agent precedence is:

1. `X-User-Agent` request header
2. `user_agent` in JSON body
3. default `go-reader/0.1`

for `/read`, set `X-User-Agent` to override the default.

## quick test

```bash
curl "http://localhost:8080/read?url=https://example.com"
```

```bash
curl -X POST "http://localhost:8080/extract" \
  -H "content-type: application/json" \
  -d '{"url":"https://example.com"}'
```

```bash
curl -X POST "http://localhost:8080/extract" \
  -H "content-type: application/json" \
  -H "X-User-Agent: my-reader/1.0" \
  -d '{"url":"https://example.com","user_agent":"body-reader/1.0"}'
```

```bash
curl "http://localhost:8080/read?url=https://example.com" \
  -H "X-User-Agent: my-reader/1.0"
```
