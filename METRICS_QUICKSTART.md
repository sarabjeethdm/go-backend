# Metrics Quick Start Guide

This guide helps you quickly get started with Prometheus metrics and monitoring for the EDI Processing System.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.25+ (for local development)

## Getting Started with Docker Compose

### 1. Start All Services

```bash
# Start MongoDB, Redis, Worker, and Prometheus
docker-compose up -d

# Verify services are running
docker-compose ps
```

### 2. Access Monitoring

**Prometheus UI:**
```bash
open http://localhost:9090
```

**API Metrics:**
```bash
curl http://localhost:8080/metrics
```

**Worker Metrics:**
```bash
curl http://localhost:9091/metrics
```

### 3. Generate Test Data

```bash
# Create some jobs to generate metrics
for i in {1..5}; do
  curl -X POST http://localhost:8080/jobs \
    -F "file=@sample.edi"
  sleep 1
done
```

### 4. View Metrics in Prometheus

1. Open http://localhost:9090
2. Click "Graph" tab
3. Try these queries:
   - `edi_jobs_total`
   - `rate(edi_jobs_total[5m])`
   - `edi_redis_queue_size`
   - `edi_active_jobs`

### 5. Check Alerts

```bash
# View alerts in browser
open http://localhost:9090/alerts

# Or via API
curl http://localhost:9090/api/v1/rules
```

## Grafana Dashboard (Optional)

### 1. Start Grafana

```bash
docker run -d \
  --name=grafana \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  grafana/grafana
```

### 2. Configure Prometheus Data Source

1. Open http://localhost:3000 (login: admin/admin)
2. Go to **Configuration** → **Data Sources**
3. Click **Add data source**
4. Select **Prometheus**
5. Set URL to:
   - Mac/Windows: `http://host.docker.internal:9090`
   - Linux: `http://172.17.0.1:9090`
6. Click **Save & Test**

### 3. Import Dashboard

1. Go to **Create** → **Import**
2. Click **Upload JSON file**
3. Select `monitoring/dashboards/grafana-dashboard.json`
4. Select **Prometheus** as data source
5. Click **Import**

### 4. View Dashboard

The dashboard will show:
- Jobs processed over time
- Job failure rate
- Processing duration (p50, p95, p99)
- Queue size
- API metrics
- And more...

## Verifying Metrics

### Check Metrics Are Being Collected

```bash
# API metrics
curl -s http://localhost:8080/metrics | grep edi_

# Worker metrics
curl -s http://localhost:9091/metrics | grep edi_
```

### Common Metrics to Check

```bash
# Job counters
curl -s http://localhost:8080/metrics | grep edi_jobs_total

# Active jobs gauge
curl -s http://localhost:9091/metrics | grep edi_active_jobs

# Queue size
curl -s http://localhost:9091/metrics | grep edi_redis_queue_size

# API request count
curl -s http://localhost:8080/metrics | grep edi_api_requests_total
```

## Example PromQL Queries

Copy these into Prometheus UI (http://localhost:9090):

### Job Metrics

```promql
# Jobs created per minute
rate(edi_jobs_total{status="created"}[1m]) * 60

# Jobs completed per minute
rate(edi_jobs_total{status="completed"}[1m]) * 60

# Job failure rate (percentage)
(rate(edi_jobs_total{status="failed"}[5m]) / 
 (rate(edi_jobs_total{status="completed"}[5m]) + rate(edi_jobs_total{status="failed"}[5m]))) * 100

# Average processing time
rate(edi_job_processing_duration_seconds_sum[5m]) / 
rate(edi_job_processing_duration_seconds_count[5m])

# 95th percentile processing time
histogram_quantile(0.95, rate(edi_job_processing_duration_seconds_bucket[5m]))
```

### API Metrics

```promql
# Total requests per second
rate(edi_api_requests_total[1m])

# Requests per second by endpoint
sum(rate(edi_api_requests_total[1m])) by (method, path)

# Error rate (5xx responses)
sum(rate(edi_api_requests_total{status=~"5.."}[5m])) / 
sum(rate(edi_api_requests_total[5m]))

# 95th percentile API response time
histogram_quantile(0.95, rate(edi_api_request_duration_seconds_bucket[5m]))
```

### Queue Metrics

```promql
# Current queue size
edi_redis_queue_size

# Queue growth rate
deriv(edi_redis_queue_size[5m])

# Active jobs
edi_active_jobs
```

## Troubleshooting

### Metrics Not Showing Up

**Check if services are running:**
```bash
docker-compose ps
```

**Check Prometheus targets:**
```bash
open http://localhost:9090/targets
```

All targets should show "UP" status.

**Check logs:**
```bash
# API logs
docker-compose logs api

# Worker logs
docker-compose logs worker

# Prometheus logs
docker-compose logs prometheus
```

### Prometheus Can't Scrape Metrics

**Docker Compose Configuration:**
- On Mac/Windows: Use `host.docker.internal` in prometheus.yml
- On Linux: Use `172.17.0.1` (Docker bridge IP)

**Verify Configuration:**
```bash
# Check Prometheus config
cat monitoring/prometheus.yml

# Restart Prometheus after config changes
docker-compose restart prometheus
```

### No Data in Grafana

1. Verify Prometheus data source is configured correctly
2. Check data source connection: **Configuration** → **Data Sources** → **Test**
3. Verify query syntax in dashboard panels
4. Check time range selector (top right in dashboard)

## Cleaning Up

### Stop All Services

```bash
# Stop all services
docker-compose down

# Remove volumes (deletes data)
docker-compose down -v
```

### Remove Grafana

```bash
docker stop grafana
docker rm grafana
```

## Next Steps

1. **Explore Alerts** - Check http://localhost:9090/alerts
2. **Customize Dashboard** - Modify panels in Grafana
3. **Add More Metrics** - Instrument custom business logic
4. **Setup Alertmanager** - Get notified of alerts
5. **Configure Remote Storage** - For long-term retention

## Additional Resources

- **Full Documentation**: [MONITORING.md](./MONITORING.md)
- **Implementation Details**: [METRICS_IMPLEMENTATION.md](./METRICS_IMPLEMENTATION.md)
- **Monitoring Config**: [monitoring/README.md](./monitoring/README.md)
- **Prometheus Docs**: https://prometheus.io/docs/
- **Grafana Docs**: https://grafana.com/docs/

## Support

For issues or questions:
1. Check [MONITORING.md](./MONITORING.md) troubleshooting section
2. Review Prometheus logs: `docker-compose logs prometheus`
3. Verify metrics endpoints are accessible: `curl http://localhost:8080/metrics`
