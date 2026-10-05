# Webhook server: test commands

## What this server does

1. A client sends a POST to `/hello` with a JSON body and an `X-Signature` header.
2. The server checks the signature, which is an HMAC-SHA256 of the raw body made with the secret key in `secret/secret.go`. A wrong or missing signature gets `401 invalid signature`.
3. If the signature is valid, the job goes onto a queue and the client gets back `{"status":"success",...}`.
4. A background worker (`job/scheduler.go`) takes jobs off the queue one at a time. For each one it calls the job's `callbackUrl` and then waits 10 seconds.

Routes:

| Route             | What it does                                              |
|-------------------|-----------------------------------------------------------|
| `/hello`          | Takes a signed webhook and queues it                      |
| `/callback-check` | Fake callback target; prints `hitting the api` in the server log |
| `/create-req`     | Signs the sample body and returns a ready-to-run curl command |

## 1. Start the server

```bash
go run .
```

Use `go run .`, not `go run main.go`. Running a single file skips the other files in the package.

## 2. Send a valid signed request (expect 200)

```bash
curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -H "X-Signature: 699d18394d280421551083c6be58f4fbc60bc31fc505c2534c88bb111d4156a4" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
```

The server log should show `Received job: hello`, then `hitting the api`, then `Finished job: hello` about 10 seconds later.

The signature only matches this exact body. If you change even one character, including spaces, you need a new signature (see step 4).

## 3. Send a request without a signature (expect 401)

```bash
curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
```

## 4. Get a fresh signature

Ask the server for one. It prints a ready-to-run curl command:

```bash
curl http://localhost:8000/create-req
```

Or sign any body yourself with openssl:

```bash
BODY='{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'
printf '%s' "$BODY" | openssl dgst -sha256 -hmac "this is a secretKey" | awk '{print $2}'
```

## 5. Load test with `hey`

Install it with `go install github.com/rakyll/hey@latest`.

```bash
hey -n 100 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -H "X-Signature: 699d18394d280421551083c6be58f4fbc60bc31fc505c2534c88bb111d4156a4" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}' \
  http://localhost:8000/hello
```

Most of these requests are expected to time out. The queue has no buffer and the worker handles one job every 10 seconds, so each request waits until the worker is free. Only the first couple will finish within hey's 20-second timeout.
