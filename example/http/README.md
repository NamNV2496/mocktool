# RESTful API

```bash
go run ./example/http/main.go
```

## Run postman

```bash
curl --location --request GET 'localhost:9090/api/v1/test' \
--header 'Content-Type: application/json' \
--data '{
  "id": 123,
  "name": "test"
}'
```

it will forward request to server

```bash
curl -L -X GET 'localhost:8081/forward/api/v1/test?feature_name=test_feature' -H 'Content-Type: application/json' -d '{"name":"test","id":123}'
```
