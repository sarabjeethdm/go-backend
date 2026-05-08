#!/bin/bash

echo "Cleaning up EDI Processing System from Kubernetes..."

kubectl delete -f k8s/07-prometheus.yaml --ignore-not-found=true
kubectl delete -f k8s/06-worker.yaml --ignore-not-found=true
kubectl delete -f k8s/05-api.yaml --ignore-not-found=true
kubectl delete -f k8s/04-redis.yaml --ignore-not-found=true
kubectl delete -f k8s/03-mongodb.yaml --ignore-not-found=true
kubectl delete -f k8s/02-pvc.yaml --ignore-not-found=true
kubectl delete -f k8s/01-configmap.yaml --ignore-not-found=true
kubectl delete -f k8s/00-namespace.yaml --ignore-not-found=true

echo "Cleanup complete!"
