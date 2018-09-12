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
    cf push $2 -f "deployment-configs/cf/manifest.yml" --vars-file="vars-file.yml"
elif [ $BIN != "none" ]; then

    echo -e "\e[93mUpdating the ConfigMap file\e[39m"
    ./update-vars

    echo -e "\e[93mApplying ConfigMap\e[39m"
    $BIN apply -f "deployment-configs/k8s/config-map.yml"

    echo -e "\e[93mApplying Secret\e[39m"
    $BIN apply -f "deployment-configs/k8s/secret.yml"

    echo -e "\e[93mProcessing and Applying cosb Template\e[39m"
    $BIN process -f "deployment-configs/k8s/template.yml" | $BIN apply -f -

    #Apply the route if we are using openshift
    if [ $BIN = "oc" ]; then

        echo -e "\e[93mApplying Route\e[39m"
        $BIN apply -f "deployment-configs/os/route.yml"
        ROUTE=$($BIN get route "cosb-route" | grep -oP "cosb-route-.*? ")
        echo -e "\e[93mURL:\e[39m http://$ROUTE"
    fi
else
    
    echo "Command '$1' not found."
    echo "Run './deploy.sh -h' for help"
fi
