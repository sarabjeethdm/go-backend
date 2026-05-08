# EDI Processing System - Monitoring Guide

This document describes the monitoring and metrics infrastructure for the EDI Processing System using Prometheus and Grafana.

## Table of Contents

- [Overview](#overview)
- [Metrics](#metrics)
- [Architecture](#architecture)
- [Setup](#setup)
- [Accessing Monitoring](#accessing-monitoring)
- [Alerts](#alerts)
- [Dashboards](#dashboards)
- [Troubleshooting](#troubleshooting)

## Overview

The EDI Processing System uses Prometheus for metrics collection and Grafana for visualization. Metrics are exposed by both the API server and worker services.

### Key Features

- Real-time metrics collection
- Historical data retention (15 days)
- Pre-configured alerts for critical conditions
- Comprehensive Grafana dashboard
- Docker Compose support for local development

## Metrics

### Exposed Endpoints

- **API Server**: `http://localhost:8080/metrics`
- **Worker**: `http://localhost:9091/metrics`
- **Prometheus**: `http://localhost:9090`

### Available Metrics

#### Job Metrics

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `edi_jobs_total` | Counter | `status` (created, processing, completed, failed) | Total number of jobs by status |
| `edi_job_processing_duration_seconds` | Histogram | - | Duration of job processing in seconds |
| `edi_active_jobs` | Gauge | - | Number of currently active/processing jobs |
| `edi_job_retries_total` | Counter | `retry_count` | Total number of job retry attempts |

#### API Metrics

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `edi_api_requests_total` | Counter | `method`, `path`, `status` | Total number of API requests |
| `edi_api_request_duration_seconds` | Histogram | `method`, `path` | Duration of API requests in seconds |

#### Queue Metrics

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `edi_redis_queue_size` | Gauge | - | Current size of the Redis job queue |

#### Transaction Metrics

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `edi_transactions_processed_total` | Counter | `transaction_type` | Total number of EDI transactions processed by type |

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│   API       │────▶│  Prometheus  │────▶│   Grafana    │
│   :8080     │     │    :9090     │     │    :3000     │
└─────────────┘     └──────────────┘     └──────────────┘
      │                     │
      │                     │
┌─────────────┐            │
│  Worker     │────────────┘
│   :9091     │
└─────────────┘
```

## Setup

### Docker Compose

The Prometheus service is already configured in `docker-compose.yml`:

```bash
# Start all services including Prometheus
docker-compose up -d

# Check Prometheus is running
curl http://localhost:9090/-/healthy

# View metrics from API
curl http://localhost:8080/metrics

# View metrics from Worker
curl http://localhost:9091/metrics
```

### Components

1. **API Server** - Exposes `/metrics` endpoint with API and job creation metrics
2. **Worker** - Runs metrics server on port 9091 with job processing metrics
3. **Prometheus** - Scrapes metrics from API and Worker services
4. **Grafana** - Visualizes metrics with pre-built dashboards (optional)

### Configuration Files

- **Docker Compose**: `monitoring/prometheus.yml`
- **Alert Rules**: `monitoring/alerts/alerts.yml`

## Accessing Monitoring

### Prometheus UI

```bash
open http://localhost:9090
```

### Example Queries

Try these queries in the Prometheus UI:

```promql
# Job processing rate
rate(edi_jobs_total{status="completed"}[5m])

# Job failure rate
rate(edi_jobs_total{status="failed"}[5m]) / 
(rate(edi_jobs_total{status="completed"}[5m]) + rate(edi_jobs_total{status="failed"}[5m]))

# 95th percentile job processing time
histogram_quantile(0.95, rate(edi_job_processing_duration_seconds_bucket[5m]))

# API request rate by endpoint
sum(rate(edi_api_requests_total[5m])) by (method, path)

# Current queue size
edi_redis_queue_size

# Active jobs count
edi_active_jobs
```

## Alerts

Alert rules are defined in `monitoring/alerts/alerts.yml`. The following alerts are configured:

### Job Alerts

| Alert | Severity | Threshold | Description |
|-------|----------|-----------|-------------|
| `HighJobFailureRate` | Warning | >10% for 5min | Job failure rate exceeds 10% |
| `CriticalJobFailureRate` | Critical | >25% for 2min | Job failure rate exceeds 25% |
| `SlowJobProcessing` | Warning | p95 >60s for 10min | Job processing is slow |
| `VerySlowJobProcessing` | Critical | p95 >120s for 5min | Job processing is very slow |
| `JobProcessingStalled` | Critical | No jobs for 10min | Job processing has stalled |

### Queue Alerts

| Alert | Severity | Threshold | Description |
|-------|----------|-----------|-------------|
| `QueueBackingUp` | Warning | >100 items for 5min | Queue is backing up |
| `QueueCriticallyBacked` | Critical | >500 items for 2min | Queue is critically backed up |
| `QueueGrowingRapidly` | Warning | >10 items/s for 5min | Queue is growing rapidly |

### API Alerts

| Alert | Severity | Threshold | Description |
|-------|----------|-----------|-------------|
| `HighAPIErrorRate` | Warning | >5% for 5min | API error rate is high |
| `CriticalAPIErrorRate` | Critical | >10% for 2min | API error rate is critical |
| `SlowAPIResponse` | Warning | p95 >1s for 5min | API response time is slow |
| `APIDown` | Critical | No requests for 5min | API is down |

### Worker Alerts

| Alert | Severity | Threshold | Description |
|-------|----------|-----------|-------------|
| `NoActiveWorkers` | Critical | 0 active, queue >0 for 5min | No workers processing jobs |
| `TooManyActiveJobs` | Warning | >50 for 10min | Too many jobs stuck processing |
| `HighRetryRate` | Warning | >0.5/s for 5min | High job retry rate |

### Viewing Alerts

**Prometheus UI:**
```
http://localhost:9090/alerts
```

You can also check active and pending alerts here.

## Dashboards

### Grafana Dashboard

A pre-configured Grafana dashboard is available in `monitoring/dashboards/grafana-dashboard.json`.

#### Dashboard Panels

1. **Jobs Processed Over Time** - Rate of jobs created, completed, and failed
2. **Job Failure Rate** - Percentage of failed jobs (gauge)
3. **Active Jobs** - Number of jobs currently processing (gauge)
4. **Job Processing Duration** - p50, p95, p99 latency percentiles
5. **Redis Queue Size** - Current queue depth
6. **API Request Rate** - Requests per second by endpoint
7. **API Response Time** - p50, p95, p99 latency percentiles
8. **EDI Transactions** - Transactions processed by type
9. **Job Retries** - Retry count distribution

#### Importing the Dashboard

1. Install Grafana (if not already installed):
   ```bash
   docker run -d -p 3000:3000 --name=grafana grafana/grafana
   ```

2. Access Grafana at `http://localhost:3000` (default credentials: admin/admin)

3. Add Prometheus as a data source:
   - Go to Configuration → Data Sources
   - Add Prometheus
   - URL: `http://prometheus:9090` (Docker) or `http://localhost:9090`
   - Save & Test

4. Import the dashboard:
   - Go to Create → Import
   - Upload `monitoring/dashboards/grafana-dashboard.json`
   - Select Prometheus data source
   - Import

### Creating Custom Dashboards

You can create custom dashboards using any of the exposed metrics. Useful visualization types:

- **Time series** - For rate metrics and trends
- **Gauge** - For current values (queue size, active jobs)
- **Heatmap** - For latency distributions
- **Table** - For detailed breakdowns by label

## Troubleshooting

### Metrics Not Appearing

1. **Check if metrics endpoints are accessible:**
   ```bash
   # API metrics
   curl http://localhost:8080/metrics
   
   # Worker metrics  
   curl http://localhost:9091/metrics
   ```

2. **Check Prometheus targets:**
   - Open `http://localhost:9090/targets`
   - Verify all targets are "UP"
   - Check for any error messages

3. **Check Prometheus logs:**
   ```bash
   # Docker Compose
   docker logs edi-prometheus
   ```

### Missing Metrics

If specific metrics are missing:

1. **Verify the code is recording metrics:**
   - Check that `metrics.RecordXXX()` calls are being made
   - Add logging around metric recording

2. **Check metric registration:**
   - Metrics are auto-registered with `promauto`
   - Ensure `metrics.Init()` is called on startup

3. **Query Prometheus directly:**
   ```bash
   curl 'http://localhost:9090/api/v1/query?query=edi_jobs_total'
   ```

### High Memory Usage

Prometheus stores metrics in memory. To reduce usage:

1. **Reduce retention time** (in docker-compose.yml or prometheus command):
   ```yaml
   --storage.tsdb.retention.time=7d
   ```

2. **Reduce scrape frequency** (in prometheus.yml):
   ```yaml
   global:
     scrape_interval: 30s
   ```

3. **Use persistent storage** (Docker volumes):
   ```yaml
   volumes:
     - prometheus-data:/prometheus
   ```

### Alert Not Firing

1. **Check alert rules are loaded:**
   ```
   http://localhost:9090/rules
   ```

2. **Verify alert expressions:**
   - Test the PromQL expression in Prometheus UI
   - Check if the condition is actually met

3. **Check alert state:**
   - `Inactive` - Condition not met
   - `Pending` - Condition met, waiting for `for` duration
   - `Firing` - Alert is active

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Alert Rule Best Practices](https://prometheus.io/docs/practices/alerting/)

## Environment Variables

### API Server

- `SERVER_PORT` - Port for API server (default: 8080)
- Metrics automatically exposed at `/metrics`

### Worker

- `METRICS_PORT` - Port for worker metrics server (default: 9091)

### Prometheus

- `--storage.tsdb.retention.time` - Data retention period (default: 15d)
- `--config.file` - Path to configuration file

## Maintenance

### Backing Up Metrics

```bash
# Create snapshot
curl -XPOST http://localhost:9090/api/v1/admin/snapshot

# Backup data directory
docker cp edi-prometheus:/prometheus ./prometheus-backup
```

### Updating Alert Rules

1. Edit `monitoring/alerts/alerts.yml`
2. Reload Prometheus configuration:
   ```bash
   # Docker Compose
   docker exec edi-prometheus killall -HUP prometheus
   
   # Or restart the container
   docker restart edi-prometheus
   ```

### Cleaning Old Data

```bash
# Delete all data
docker exec edi-prometheus rm -rf /prometheus/*

# Restart Prometheus
docker restart edi-prometheus
```
