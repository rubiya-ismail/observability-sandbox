## Prometheus

### TODO

- Add an observability-network to all services - user, order, gateway
- Add a name and host to each docker command
- Test of gateway can call user or order
  - Add a user
  - Display all users

### Start prometheus

- Go to root of repo containing folders like "monitoring", "services", etc
- Run the following command:

```
$>  docker run -p 9090:9090 -v ${PWD}/monitoring/prometheus/prometheus-dev.yml:/etc/prometheus/prometheus.yml prom/prometheus
```

### Access the Web UI

- Open your browser: http://localhost:9090
- From here you can:
  - Query metrics using PromQL
  - View graphs for specific metrics
  - Check targets (Status → Targets) to see which services Prometheus is scraping
  - Inspect configuration (Status → Configuration) to verify your scrape jobs

### Other useful commands

- List all active targets
  ```
  $> docker exec -it prometheus curl http://localhost:9090/api/v1/targets
  ```
- Example: query gateway request metrics
  ```
  $> curl 'http://localhost:9090/api/v1/query?query=http_requests_total'
  ```

### Prometheus Query Language lets you explore metrics:

- You can do sums, averages, percentiles, etc., and visualize them in the Web UI.
- Example: total HTTP requests for gateway in last 5 minutes

  ```
  rate(http_requests_total{service="gateway"}[5m])

  ```

### Check Logs & Debug

Useful to verify that Prometheus is able to scrape all targets.

```
$> docker logs prometheus
```

### Integrate with Grafana

- Point Grafana to Prometheus as a data source.
- Build dashboards for metrics like:
  - Request rates
  - Latencies
  - Error counts
