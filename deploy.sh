#!/bin/bash
if [ $# = 0 ] || [ $1 = "-h" ] || [ $1 = "--help" ]; then
    echo -e "\e[93mCloudFoundry\e[39m deployment: ./deploy.sh cf [APP_NAME]"
    echo -e "\e[93mKubernetes\e[39m deployment: ./deploy.sh k8s"
    exit
fi

if [ $1 = "cf" ]; then
    cf push $1 -f "manifest.yml" --vars-file="vars-file.yml"
elif [ $1 = "k8s" ]; then
    echo -e "\e[93mUpdating ConfigMap\e[39m"
    kubectl apply -f "k8s-configMap.yaml"

    echo -e "\e[93mUpdating Deployment\e[39m"
    kubectl apply -f "k8s-deployment.yaml"

    echo -e "\e[93mUpdating Deployment\e[39m"
    kubectl apply -f "k8s-service.yaml"
else
    echo "Command '$1' not found."
    echo "Run './deploy.sh -h' for help"
fi
