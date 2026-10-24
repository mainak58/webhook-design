# webhook

A small webhook server in Go, built as a learning project. It accepts signed requests, queues them, and calls the requester back at a URL they choose, retrying with backoff when the callback fails. It uses only the standard library.

It is not production-ready: jobs live in memory, so a restart loses them. See [Limitations](#limitations).

## How it works

1. A client sends `POST /hello` with a JSON body and an `X-Signature` header. The signature is the hex HMAC-SHA256 of the raw body, made with the shared secret in `secret/secret.go`.
2. The server checks the signature (`401` if it's wrong or missing) and parses the JSON (`400` if it's invalid).
3. The job goes onto an in-memory queue that holds 1000 jobs. The client gets `202 Accepted` with a job ID, or `503` if the queue is full.
4. A pool of 10 workers takes jobs off the queue and sends a `GET` to each job's `callbackUrl`. Any `2xx` response counts as success.
5. If a callback fails, the job is retried after 1, 2, 4 and 8 minutes, each with up to 10% random jitter. After 5 failed attempts it is given up on.

```
client ──POST /hello──► verify HMAC ──► queue (1000) ──► 10 workers ──GET──► callbackUrl
                                          ▲                   │
                                          └── retry timer ◄───┘ on failure
```

### Request

```http
POST /hello
Content-Type: application/json
X-Signature: <hex HMAC-SHA256 of the raw body>

{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}
```

### Response

`202 Accepted`, `Content-Type: application/json`:

```json
{"status":"accepted","message":"webhook queued","id":"<server-generated UUID>"}
```

### Routes

| Route             | What it does |
|-------------------|--------------|
| `/hello`          | Takes a signed webhook and queues it |
| `/callback-check` | Fake callback target for local testing; prints `hitting the api` in the server log |
| `/create-req`     | Development helper: signs a sample body and returns a ready-to-run curl command |

### Code layout

| Path | Role |
|------|------|
| `main.go` | Starts the workers and the HTTP server on `:8000` |
| `queue/queue.go` | The `/hello` handler and the job queue |
| `job/scheduler.go` | Worker pool, delivery and retry with backoff |
| `job/utils.go` | The `/create-req` helper |
| `secret/secret.go` | HMAC-SHA256 signing and verification |

## Running it

Requires Go 1.27 or later.

```bash
go run .
```

Use `go run .`, not `go run main.go`. Running a single file skips the other files in the package.

The server listens on `http://localhost:8000`.

## Trying it out

### Send a valid signed request (expect 202)

```bash
curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -H "X-Signature: 699d18394d280421551083c6be58f4fbc60bc31fc505c2534c88bb111d4156a4" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
```

You get `202` with a job ID, and the server log shows `hitting the api` when the callback is delivered.

The signature only matches this exact body. If you change even one character, including spaces, you need a new signature (see below).

### Send a request without a signature (expect 401)

```bash
curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
```

### Watch a failing callback get retried

Point `callbackUrl` at a port where nothing is listening:

```bash
BODY='{"request_id":"test","message":"bad","callbackUrl":"http://localhost:1/x"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "this is a secretKey" | awk '{print $2}')
curl -i -X POST http://localhost:8000/hello -H "X-Signature: $SIG" -d "$BODY"
```

The log shows `bad attempt 1: error making request: ...` straight away, then attempts 2 to 5 after about 1, 2, 4 and 8 more minutes.

### Get a fresh signature

Ask the server for one. It prints a ready-to-run curl command:

```bash
curl http://localhost:8000/create-req
```

Or sign any body yourself with openssl:

```bash
BODY='{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
printf '%s' "$BODY" | openssl dgst -sha256 -hmac "this is a secretKey" | awk '{print $2}'
```

### Load test with `hey`

Install it with `go install github.com/rakyll/hey@latest`.

```bash
hey -n 1000 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -H "X-Signature: 699d18394d280421551083c6be58f4fbc60bc31fc505c2534c88bb111d4156a4" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}' \
  http://localhost:8000/hello
```

The handler only queues the job, so requests should return `202` quickly. Once 1000 jobs are waiting in the queue, further requests get `503` until the workers catch up.

## Limitations

What would need fixing before real use:

- Jobs are kept in memory. A crash or restart loses every queued job and every job waiting for a retry, even though the client already got `202`. Storing jobs in a database table would fix this.
- The callback is an unsigned `GET` with no body, so the receiver can't tell which job finished or check that the call came from this server.
- Any URL is accepted as `callbackUrl`, including internal addresses (SSRF).
- There is no replay protection. A captured signed request can be resent forever, and `request_id` isn't used to drop duplicates.
- The secret is hardcoded and shared by all clients. Don't reuse it anywhere.
- `/create-req` signs requests for anyone, so it's for local development only.
- There is no limit on request body size and no server timeout.
