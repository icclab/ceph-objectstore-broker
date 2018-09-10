#!/bin/bash

echo -e "\e[93mUpdating ConfigMap\e[39m"
kubectl apply -f "k8s-configMap.yaml"

echo -e "\e[93mUpdating Deployment\e[39m"
kubectl apply -f "k8s-deployment.yaml"