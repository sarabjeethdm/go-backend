# Monitoring Configuration

This directory contains monitoring and alerting configuration for the EDI Processing System.

## Directory Structure

```
monitoring/
├── prometheus.yml          # Prometheus configuration for Docker Compose
├── alerts/
│   └── alerts.yml         # Prometheus alert rules
└── dashboards/
    └── grafana-dashboard.json  # Pre-configured Grafana dashboard
```

## Files

### prometheus.yml

Prometheus scrape configuration for Docker Compose environment. Defines:
- Scrape intervals and evaluation rules
- Job configurations for API and Worker services
- Alert rule loading

### alerts/alerts.yml

Comprehensive alert rules including:
- **Job Alerts**: Failure rates, processing delays, stalled processing
- **Queue Alerts**: Queue backup, rapid growth
- **API Alerts**: Error rates, response times, downtime
- **Worker Alerts**: No active workers, stuck jobs, high retry rates

### dashboards/grafana-dashboard.json

Ready-to-import Grafana dashboard with 9 panels:
1. Jobs processed over time
2. Job failure rate gauge
3. Active jobs gauge
4. Job processing duration (p50, p95, p99)
5. Redis queue size
6. API request rate by endpoint
7. API response time (p50, p95, p99)
8. EDI transactions by type
9. Job retries by count

## Usage

### With Docker Compose

The `prometheus.yml` file is automatically mounted when you run:

```bash
docker-compose up -d
```

Access Prometheus at: http://localhost:9090

### With Kubernetes

For Docker Compose deployments, the configuration is automatically loaded from `prometheus.yml`

### Importing Grafana Dashboard

1. Start Grafana:
   ```bash
   docker run -d -p 3000:3000 --name=grafana grafana/grafana
   ```

2. Open http://localhost:3000 (credentials: admin/admin)

3. Add Prometheus data source:
   - Configuration → Data Sources → Add data source
   - Select Prometheus
   - URL: `http://prometheus:9090` or `http://localhost:9090`
   - Save & Test

4. Import dashboard:
   - Create → Import
   - Upload `dashboards/grafana-dashboard.json`
   - Select Prometheus data source
   - Import

## Customization

### Adding New Alerts

Edit `alerts/alerts.yml` and add your alert rule:

```yaml
- alert: YourAlertName
  expr: your_promql_expression > threshold
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Alert summary"
    description: "Alert description"
```

Reload Prometheus configuration:
```bash
docker exec edi-prometheus killall -HUP prometheus
```

### Modifying Scrape Targets

Edit `prometheus.yml` scrape_configs section to add or modify targets.

### Dashboard Customization

1. Import the dashboard to Grafana
2. Make your changes in the Grafana UI
3. Export the updated dashboard
4. Save to `dashboards/grafana-dashboard.json`

## Testing

### Verify Prometheus Targets

```bash
curl http://localhost:9090/api/v1/targets
```

### Check Alert Rules

```bash
curl http://localhost:9090/api/v1/rules
```

### Query Metrics

```bash
curl 'http://localhost:9090/api/v1/query?query=edi_jobs_total'
```

## Documentation

For detailed documentation, see [../MONITORING.md](../MONITORING.md)
