# Mock response data for grpc

```bash
# start client
go run example/grpc/main.go
```

CURL
use gRPC or http request

```bash
curl -L -X GET 'http://localhost:8080/api/v1/events' -H 'Content-Type: application/json' -d '{"limit":20,"offset":1}'
```

