# Overview
--

This observability sandbox project is a solid choice for demonstrating modern backend practices. Let me outline a practical implementation that leverages your Go and containerization expertise.

## Project Structure

```
observability-sandbox/
├── docker-compose.yml
├── services/
│   ├── api-gateway/
│   ├── user-service/
│   └── order-service/
├── monitoring/
│   ├── prometheus/
│   ├── grafana/
│   └── jaeger/
└── shared/
    └── telemetry/
```

## Core Implementation Strategy

**Service Architecture**: Three Go microservices (API Gateway, User Service, Order Service) with realistic inter-service communication patterns. Each service will implement proper instrumentation for metrics, logs, and traces.

**Observability Stack**:
- Prometheus for metrics collection
- Grafana for visualization dashboards  
- Jaeger for distributed tracing
- Structured logging with correlation IDs

**Key Go Libraries**:
- `gin-gonic/gin` for HTTP routing
- `prometheus/client_golang` for metrics
- `opentelemetry-go` for tracing
- `logrus` for structured logging

Would you like me to start with the Docker Compose configuration and shared telemetry package, or would you prefer to begin with one of the microservices? The shared telemetry package would establish consistent instrumentation patterns across all services, following DDD principles for clean separation of concerns.

Also, any specific business domain you'd prefer for the services (e-commerce, user management, etc.), or shall we stick with the user/order example for simplicity?
