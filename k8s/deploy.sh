#!/bin/bash

set -e

echo "========================================="
echo "EDI Processing System - Kubernetes Setup"
echo "========================================="
echo ""

echo "Step 1: Building Docker images..."
echo "Building API image..."
docker build -t golang-backend-task-api:latest -f Dockerfile .

echo "Building Worker image..."
docker build -t golang-backend-task-worker:latest -f Dockerfile.worker .

echo ""
echo "Step 2: Loading images into Kubernetes..."

if command -v minikube &> /dev/null; then
    echo "Loading images into Minikube..."
    minikube image load golang-backend-task-api:latest
    minikube image load golang-backend-task-worker:latest
elif command -v kind &> /dev/null; then
    CLUSTER_NAME=${KIND_CLUSTER_NAME:-kind}
    echo "Loading images into Kind cluster ($CLUSTER_NAME)..."
    kind load docker-image golang-backend-task-api:latest --name $CLUSTER_NAME
    kind load docker-image golang-backend-task-worker:latest --name $CLUSTER_NAME
else
    echo "Using Docker Desktop Kubernetes - images already available"
fi

echo ""
echo "Step 3: Creating namespace and resources..."
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/01-configmap.yaml
kubectl apply -f k8s/02-pvc.yaml

echo ""
echo "Step 4: Deploying MongoDB and Redis..."
kubectl apply -f k8s/03-mongodb.yaml
kubectl apply -f k8s/04-redis.yaml

echo "Waiting for MongoDB to be ready..."
kubectl wait --for=condition=ready pod -l app=mongodb -n edi-processing --timeout=120s

echo "Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis -n edi-processing --timeout=120s

echo ""
echo "Step 5: Deploying API and Worker services..."
kubectl apply -f k8s/05-api.yaml
kubectl apply -f k8s/06-worker.yaml

echo "Waiting for API to be ready..."
kubectl wait --for=condition=ready pod -l app=edi-api -n edi-processing --timeout=120s

echo "Waiting for Worker to be ready..."
kubectl wait --for=condition=ready pod -l app=edi-worker -n edi-processing --timeout=120s

echo ""
echo "Step 6: Deploying Prometheus..."
kubectl apply -f k8s/07-prometheus.yaml

echo ""
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
echo "To access the services, run:"
echo ""
echo "  API Server:    kubectl port-forward -n edi-processing svc/edi-api 8080:8080"
echo "  Worker Metrics: kubectl port-forward -n edi-processing svc/edi-worker 9091:9091"
echo "  Prometheus:    kubectl port-forward -n edi-processing svc/prometheus 9090:9090"
echo ""
echo "Check status with:"
echo "  kubectl get pods -n edi-processing"
echo "  kubectl get svc -n edi-processing"
echo ""
