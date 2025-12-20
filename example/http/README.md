# RESTful API

```bash
go run ./example/http/main.go
```

## run postman

```bash
curl --location --request GET 'localhost:9090/api/v1/test' \
--header 'Content-Type: application/json' \
--data '{
  "id": 123,
  "name": "test"
}'
```

