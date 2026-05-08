# Kubernetes Deployment Guide

This guide provides complete instructions for deploying the EDI Processing System on Kubernetes (Minikube, Kind, or Docker Desktop Kubernetes).

## Prerequisites

### Required Tools
- **kubectl**: Kubernetes command-line tool
- **Docker**: For building container images
- **One of the following**:
  - Minikube
  - Kind (Kubernetes in Docker)
  - Docker Desktop with Kubernetes enabled

### Verify Installation
```bash
kubectl version --client
docker --version

# For Minikube
minikube version

# For Kind
kind version

# For Docker Desktop
kubectl config get-contexts
```

## Quick Start

### Option 1: Automated Deployment (Recommended)

```bash
# Navigate to project directory
cd golang-backend-task

# Run deployment script
./k8s/deploy.sh
```

The script will:
1. Build Docker images for API and Worker
2. Load images into your Kubernetes cluster
3. Deploy all resources in order
4. Wait for services to be ready

### Option 2: Manual Deployment

```bash
# Step 1: Build images
docker build -t golang-backend-task-api:latest -f Dockerfile .
docker build -t golang-backend-task-worker:latest -f Dockerfile.worker .

# Step 2: Load images into cluster
# For Minikube:
minikube image load golang-backend-task-api:latest
minikube image load golang-backend-task-worker:latest

# For Kind:
kind load docker-image golang-backend-task-api:latest
kind load docker-image golang-backend-task-worker:latest

# For Docker Desktop: Images are already available

# Step 3: Deploy resources
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/01-configmap.yaml
kubectl apply -f k8s/02-pvc.yaml
kubectl apply -f k8s/03-mongodb.yaml
kubectl apply -f k8s/04-redis.yaml
kubectl apply -f k8s/05-api.yaml
kubectl apply -f k8s/06-worker.yaml
kubectl apply -f k8s/07-prometheus.yaml
```

## Accessing the Services

### Port Forwarding

The services are deployed as ClusterIP (internal only). Use port-forwarding to access them:

```bash
# API Server (in one terminal)
kubectl port-forward -n edi-processing svc/edi-api 8080:8080

# Worker Metrics (in another terminal)
kubectl port-forward -n edi-processing svc/edi-worker 9091:9091

# Prometheus (in another terminal)
kubectl port-forward -n edi-processing svc/prometheus 9090:9090

# MongoDB (for debugging)
kubectl port-forward -n edi-processing svc/edi-mongodb 27017:27017

# Redis (for debugging)
kubectl port-forward -n edi-processing svc/edi-redis 6379:6379
```

### Testing the API

Once port-forwarding is active:

```bash
# Health check
curl http://localhost:8080/health

# Create a job
echo "CLAIM*CLM001*MEM123*2500" > test.edi
curl -X POST http://localhost:8080/jobs -F "file=@test.edi" -F "format=X12"

# Check job status (replace {job-id} with actual ID)
curl http://localhost:8080/jobs/{job-id}

# Get job result
curl http://localhost:8080/jobs/{job-id}/result
```

## Monitoring and Debugging

### Check Pod Status
```bash
kubectl get pods -n edi-processing
kubectl get pods -n edi-processing -w  # Watch mode
```

### View Logs
```bash
# API logs
kubectl logs -n edi-processing -l app=edi-api -f

# Worker logs
kubectl logs -n edi-processing -l app=edi-worker -f

# MongoDB logs
kubectl logs -n edi-processing -l app=mongodb -f

# Redis logs
kubectl logs -n edi-processing -l app=redis -f

# Prometheus logs
kubectl logs -n edi-processing -l app=prometheus -f
```

### Describe Resources
```bash
kubectl describe pod -n edi-processing -l app=edi-api
kubectl describe deployment -n edi-processing edi-api
kubectl describe service -n edi-processing edi-api
```

### Get All Resources
```bash
kubectl get all -n edi-processing
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│               Kubernetes Cluster                        │
│  ┌───────────────────────────────────────────────────┐ │
│  │         Namespace: edi-processing                 │ │
│  │                                                   │ │
│  │  ┌─────────────┐      ┌─────────────┐          │ │
│  │  │   API Pod   │──┬──▶│ MongoDB Pod │          │ │
│  │  │  (2 replicas)│  │   │   (1 replica) │          │ │
│  │  └─────────────┘  │   └─────────────┘          │ │
│  │         │          │          │                 │ │
│  │         │          │    ┌─────────────┐        │ │
│  │         │          └───▶│  Redis Pod  │        │ │
│  │         │               │   (1 replica) │        │ │
│  │         │               └─────────────┘        │ │
│  │         │                      ▲                │ │
│  │  ┌─────────────┐              │                │ │
│  │  │ Worker Pod  │──────────────┘                │ │
│  │  │  (2 replicas)│                               │ │
│  │  └─────────────┘                               │ │
│  │                                                 │ │
│  │  ┌─────────────┐                               │ │
│  │  │ Prometheus  │───── Scrapes Metrics          │ │
│  │  │    Pod      │                               │ │
│  │  └─────────────┘                               │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## Resource Specifications

### Deployments

| Service | Replicas | Image | Resources |
|---------|----------|-------|-----------|
| API | 2 | golang-backend-task-api:latest | 128Mi-512Mi / 100m-500m |
| Worker | 2 | golang-backend-task-worker:latest | 128Mi-512Mi / 100m-500m |
| MongoDB | 1 | mongo:7.0 | 256Mi-512Mi / 250m-500m |
| Redis | 1 | redis:7.2-alpine | 128Mi-256Mi / 100m-200m |
| Prometheus | 1 | prom/prometheus:v2.50.1 | 256Mi-512Mi / 100m-500m |

### Persistent Volumes

| Name | Size | Used By |
|------|------|---------|
| mongodb-pvc | 5Gi | MongoDB |
| redis-pvc | 1Gi | Redis |
| prometheus-pvc | 2Gi | Prometheus |

### Services

| Name | Type | Port | Target Port |
|------|------|------|-------------|
| edi-api | ClusterIP | 8080 | 8080 |
| edi-worker | ClusterIP | 9091 | 9091 |
| edi-mongodb | ClusterIP | 27017 | 27017 |
| edi-redis | ClusterIP | 6379 | 6379 |
| prometheus | ClusterIP | 9090 | 9090 |

## Configuration

Application configuration is managed via ConfigMap (`edi-config`):

```yaml
MONGODB_URI: "mongodb://edi-mongodb:27017"
MONGODB_DATABASE: "edi_processor"
REDIS_HOST: "edi-redis"
REDIS_PORT: "6379"
LOG_LEVEL: "info"
WORKER_MAX_RETRIES: "3"
WORKER_POLL_INTERVAL: "1"
WORKER_INITIAL_BACKOFF: "2"
WORKER_SHUTDOWN_TIMEOUT: "30"
METRICS_PORT: "9091"
```

To update configuration:
```bash
kubectl edit configmap edi-config -n edi-processing
# Then restart pods
kubectl rollout restart deployment -n edi-processing
```

## Scaling

### Scale API Service
```bash
kubectl scale deployment edi-api -n edi-processing --replicas=3
```

### Scale Worker Service
```bash
kubectl scale deployment edi-worker -n edi-processing --replicas=5
```

### Horizontal Pod Autoscaler (HPA)
```bash
# Enable HPA for API (requires metrics-server)
kubectl autoscale deployment edi-api -n edi-processing \
  --cpu-percent=70 --min=2 --max=10
```

## Health Checks

All deployments include health checks:

### API Service
- **Liveness**: `GET /health` every 10s
- **Readiness**: `GET /health` every 5s

### Worker Service
- **Liveness**: `GET /metrics` every 10s
- **Readiness**: `GET /metrics` every 5s

### MongoDB
- **Liveness**: `mongosh --eval db.adminCommand('ping')`
- **Readiness**: Same as liveness

### Redis
- **Liveness**: `redis-cli ping`
- **Readiness**: Same as liveness

## Cleanup

### Remove All Resources

```bash
# Using cleanup script
./k8s/cleanup.sh

# Or manually
kubectl delete namespace edi-processing
```

## Troubleshooting

### Pods Not Starting

```bash
# Check events
kubectl get events -n edi-processing --sort-by='.lastTimestamp'

# Check pod status
kubectl describe pod <pod-name> -n edi-processing

# Check logs
kubectl logs <pod-name> -n edi-processing
```

### Image Pull Errors

If you see `ImagePullBackOff`:
- For Minikube: Run `minikube image load <image-name>`
- For Kind: Run `kind load docker-image <image-name>`
- For Docker Desktop: Ensure images are built locally

### PVC Pending

```bash
# Check PVC status
kubectl get pvc -n edi-processing

# For Minikube, ensure storage provisioner is enabled
minikube addons enable storage-provisioner
```

### Service Connection Issues

```bash
# Test DNS resolution
kubectl run -it --rm debug --image=busybox -n edi-processing -- nslookup edi-mongodb

# Test connectivity
kubectl run -it --rm debug --image=busybox -n edi-processing -- wget -O- http://edi-api:8080/health
```

## Production Considerations

For production deployments, consider:

1. **Ingress Controller**: Replace port-forward with proper Ingress
2. **TLS/SSL**: Add certificates for secure communication
3. **Resource Limits**: Adjust based on load testing
4. **Persistent Storage**: Use appropriate storage classes
5. **Backup Strategy**: Regular backups of MongoDB data
6. **Monitoring**: Full Prometheus + Grafana stack
7. **Secrets Management**: Use Kubernetes Secrets for sensitive data
8. **Network Policies**: Restrict pod-to-pod communication
9. **Pod Disruption Budgets**: Ensure high availability
10. **Affinity Rules**: Spread pods across nodes

## Next Steps

1. Set up Ingress for external access
2. Configure Grafana dashboards
3. Set up automated backups
4. Implement CI/CD pipeline
5. Configure log aggregation (ELK/Loki)
