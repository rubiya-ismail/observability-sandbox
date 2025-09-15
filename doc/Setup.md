## Go Module Setup

Let's initialize the Go modules properly. Run these commands in your project root:

### 1. Initialize Main Module
```bash
# In observability-sandbox/
go mod init observability-sandbox
```

### 2. Initialize Service Modules
```bash
# User Service
cd services/user-service
go mod init user-service
cd ../..

# Order Service  
cd services/order-service
go mod init order-service
cd ../..

# API Gateway
cd services/api-gateway
go mod init api-gateway
cd ../..
```

### 3. Create Shared Telemetry Module
```bash
# Shared telemetry
cd shared/telemetry
go mod init telemetry
cd ../..
```

### 4. Add Dependencies to Telemetry Module
```bash
cd shared/telemetry
go get github.com/prometheus/client_golang/prometheus
go get github.com/sirupsen/logrus
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/jaeger
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/semconv/v1.4.0
cd ../..
```

### 5. Verify Setup
Check that you have these files created:
- `observability-sandbox/go.mod`
- `services/user-service/go.mod`
- `services/order-service/go.mod` 
- `services/api-gateway/go.mod`
- `shared/telemetry/go.mod`

The telemetry module should show the dependencies we just added in its `go.mod` file.

**Ready for next step?** Once you confirm the modules are set up correctly, we can create the telemetry package structure and define the interfaces.
