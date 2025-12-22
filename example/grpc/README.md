# Mock response data for grpc

```bash
# Generate protobuf 

# Way 1: not recommend

git clone https://github.com/googleapis/googleapis.git ~/googleapis

protoc -I example/grpc/proto -I ~/googleapis --go_out=example/grpc/proto/generated --go-grpc_out=example/grpc/proto/generated example/grpc/proto/test.proto

# Way 2

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest


export PATH="/Users/namnv/go/bin:$PATH" && cd /Users/namnv/Workspace/GIT/mockTool/example/grpc/proto && buf generate
```


```bash
# start client
go run example/grpc/main.go
```

CURL
use gRPC or http request

```bash
curl -L -X GET 'http://localhost:8080/api/v1/test' -H 'Content-Type: application/json' -d '{"id":123,"name":"test"}'
```

