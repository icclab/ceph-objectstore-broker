#!/bin/bash

echo "Updating ConfigMap"
kubectl apply -f "k8s-configMap.yaml"

echo "Updating Deployment"
kubectl apply -f "k8s-deployment.yaml"