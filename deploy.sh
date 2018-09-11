#!/bin/bash
if [ $# = 0 ] || [ $1 = "-h" ] || [ $1 = "--help" ]; then
    echo -e "\e[93mCloudFoundry\e[39m deployment: ./deploy.sh cf [APP_NAME]"
    echo -e "\e[93mKubernetes\e[39m deployment: ./deploy.sh k8s"
    echo -e "\e[93mOpenShift\e[39m deployment: ./deploy.sh os"
    exit
fi

# Choose between k8s or openshift
BIN="none"
if [ $1 = "k8s" ]; then
    BIN="kubectl"  
elif [ $1 = "os" ]; then
    BIN="oc"
fi

if [ $1 = "cf" ]; then
    if [ $# -lt 2 ]; then
        echo -e "Please provide the \e[93mAPP_NAME\e[39m to be used for the deployment."
        exit
    fi
    cf push $2 -f "manifest.yml" --vars-file="vars-file.yml"
elif [ $BIN != "none" ]; then
    echo -e "\e[93mUpdating ConfigMap\e[39m"
    $BIN apply -f "configMap.yml"

    echo -e "\e[93mUpdating Deployment\e[39m"
    $BIN apply -f "deployment.yml"

    echo -e "\e[93mUpdating Deployment\e[39m"
    $BIN apply -f "service.yml"

    #Apply the route if we are using openshift
    if [ $BIN = "oc" ]; then
        echo -e "\e[93mUpdating Route\e[39m"
        $BIN apply -f "route.yml"
    fi
else
    echo "Command '$1' not found."
    echo "Run './deploy.sh -h' for help"
fi
