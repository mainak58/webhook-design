hey -n 100 -c 100   -m POST   -H "Content-Type: application/json"   -d '{"request_id":"test", "message":"hello"}'   http://localhost:8000/hello




curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -d '{"request_id":"test","message":"hello"}'


  curl -i -X POST http://localhost:8000/hello \
  -H "Content-Type: application/json" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}'


hey -n 100 -c 100 \
  -m POST \
  -H "Content-Type: application/json" \
  -d '{"request_id":"test","message":"hello","callbackUrl":"http://localhost:8000/callback-check"}' \
  http://localhost:8000/hello