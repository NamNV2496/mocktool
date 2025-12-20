# Mocktool

Mocktool is simple tool written by golang. It support control response of API by scenario of feature.

At the same time we have:
- Multiple active feature
- Only 1 active scenario for each feature
- Multiple API for each scenario (same API path but different requestBody)

![doc/1.png](doc/architect.png)


# Multiple active feature at the same time

The feature_name is control by `api-service`. Which support multiple services work at the same time.
```go
// Service 1
	targetURL := "http://localhost:8081/forward" + c.Request().RequestURI + "?feature_name=test_feature"

// Service 2
	targetURL := "http://localhost:8081/forward" + c.Request().RequestURI + "?feature_name=featur2"
```

![doc/1.png](doc/1.png)

# Only 1 active scenario for each feature

- If I add new scenario of a feature, that new scenario will active and deactive others.
- If I active a existed scenario of a feature. others scenarios will deactive

=> Make sure 1 API can response expecting answer

![doc/2.png](doc/2.png)
![doc/2.png](doc/3.png)

# Multiple API for each scenario 


![doc/6.png](doc/6.png)
![doc/7.png](doc/7.png)
![doc/8.png](doc/8.png)

## result

![doc/9.png](doc/9.png)
![doc/10.png](doc/10.png)


