# How to run the service ?

## Build Docker Image

```
$ cd <root of project, folder containing services, and shared folder>
$ docker build --no-cache --progress=plain  -f services/api-gateway/Dockerfile -t api-gateway:latest .
```

## Create Shared Docker Network

```
$ docker network create observablity-net
```

## Run the Service

- Running service with a "service name" in "shared docker network"

```
$ docker run --name api-gateway --network observablity-net -p 8081:8081 api-gateway:latest
```

- Running service in detached mode

```
$ docker run -d --name api-gateway --network observablity-net -p 8081:8081 api-gateway:latest
```

- To re-run the above commands, remove (or rename) the 'registered' service name as follows:

```
$ docker rm api-gateway
```

## Stop the Service

```
$ docker stop api-gateway
```

# API Design

## API Gateway

| Method   | Route              | Purpose                           | Forwarding Rule                                |
| -------- | ------------------ | --------------------------------- | ---------------------------------------------- |
| GET      | `/health`          | Gateway health check              | Handled by gateway                             |
| GET      | `/metrics`         | Gateway metrics (Prometheus etc.) | Handled by gateway                             |
| GET/POST | `/users`           | User-related API requests         | Forward to **service_endpoint** in config file |
| GET/POST | `/orders`          | Order-related API requests        | Forward to **service_endpoint** in config file |
| GET      | `/services/health` | Aggregated health of all services | Gateway queries registered services            |

## Configuration

The 'services' map contains a map of service name to attributes.
The '\*\_endpoint' attributes has the path to the service URL, like, "/users"
This attribute may contain the version of the downstream API, like, "/api/v1/users", or "/api/v2/orders".

### Sample Configuration

| Method | Route                                  | Forwarding URL                                                                  | Notes                                                  |
| ------ | -------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------ |
| GET    | `http://locahost:8081/users`           | http://user-service:8082/api/v1/users                                           | If not from gateway, use localhost instead of base_url |
| GET    | `http://locahost:8081/orders`          | http://order-service:8082/api/v1/orders                                         | If not from gateway, use localhost instead of base_url |
| GET    | `http://locahost:8081/services/health` | http://user-service:8082/api/v1/health, http://order-service:8082/api/v1/health | See health_endpoint                                    |

```
{
  "environment": "dev",
  "services": {
    "users": {
      "name": "user-service",
      "base_url": "http://user-service:8082",
      "api_version": "v1",
      "timeout": "30s",
      "retries": 3,
      "service_endpoint": "/api/v1/users",
      "health_endpoint": "/api/v1/health",
      "metrics_endpoint": "/api/v1/metrics"
    },
    "orders": {
      "name": "order-service",
      "base_url": "http://order-service:8083",
      "api_version": "v1",
      "timeout": "30s",
      "retries": 3,
      "service_endpoint": "/api/v1/orders",
      "health_endpoint": "/api/v1/health",
      "metrics_endpoint": "/api/v1/metrics"
    }
  },
  "rate_limits": {
    "requests_per_minute": 1000,
    "window_size": "1m"
  },
  "gateway": {
    "port": 8081,
    "read_timeout": "30s",
    "write_timeout": "30s"
  }
}
```
