# RESTful API

```bash
go run ./example/http/main.go
```

## Run postman

```bash
curl --location --request GET 'localhost:8080/api/v1/test' \
--header 'Content-Type: application/json' \
--data '{
  "id": 123,
  "name": "test"
}'
```

it will forward request to server

```bash
curl -L -X GET 'localhost:8081/forward/api/v1/test?feature_name=test_feature' -H 'Content-Type: application/json' -H 'X-Account-Id: 1' -H 'X-Feature-Name: test_feature' -d '{"name":"test","id":123}'
```

## Query param


```bash
curl -L -X GET 'http://localhost:8080/api/v1/test?is_sort=true&is_new=true' -H 'Content-Type: application/json' -d '{"id":123,"name":"test"}'
```

```bash
curl -L -X GET 'localhost:8081/forward/api/v1/test?is_sort=true&is_new=true' -H 'Content-Type: application/json' -H 'X-Account-Id: 1' -H 'X-Feature-Name: test_feature' -d '{"id":123,"name":"test"}'
```

