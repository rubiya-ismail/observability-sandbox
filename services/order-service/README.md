# Build docker image

```
$> cd <root of project, folder containing services, and shared folder>
$> docker build --no-cache --progress=plain  -f services/order-service/Dockerfile -t order-service:latest .
```

# Run service

```
$> docker network create observablity-net

$> docker run -d --name order-service --network observablity-net -p 8083:8083 order-service:latest
```
