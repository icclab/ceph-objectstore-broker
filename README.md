# Service Broker for a Ceph Object Store

This broker is compliant with V2 of the [Open Service Broker API](https://www.openservicebrokerapi.org/). The broker provides access to a [Ceph](https://ceph.com/) object store,
and can currently be deployed as CloudFoundry app, on Kubernetes or on OpenShift. Deployment as a Bosh release is planned for the future.

## Table of Contents

* [General Operation](#General-Operation)
* [Deployment](#Deployment)
  * [Prerequisites](#Prerequisites)
  * [CloudFoundry](#CloudFoundry)
  * [Bosh Release](#Bosh-Release)
  * [Kubernetes & OpenShift](#Kubernetes-&-OpenShift)
* [Integration Tests](#Integration-Tests)

## General Operation

When an instance is [provisioned](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#provisioning) a user is created on Ceph. Then when an
application [binds](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#binding) to the broker, it returns access credentials for both the S3 and Swift
APIs supported by Ceph.

The credentials made available to the application (through environment variables) after a bind are:-

* s3User
* s3AccessKey
* s3SecretKey
* s3Endpoint
* swiftUser
* swiftSecretKey
* swiftEndpoint

## Deployment

Deployment to all platforms is done through the ```deploy.sh``` file, so once prerequisites for a platform are fulfilled the script can be used to deploy the broker.

### Prerequisites

Before deploying to a platform, you need to provide the required details about your Ceph installation. Specifically you will need a
[Ceph object gateway](http://docs.ceph.com/docs/master/radosgw/) setup. The broker will use the admin user on the gateway to manage users there as required to operate the
service, and so it requires a number of variables including the gateway's endpoint and access keys for the user.

To provide the required information you will need a file called ```vars-file.yml```. A template for this file called ```vars-file-template.yml``` is available, and so can simply
be copied, renamed and then the details filled in.

### CloudFoundry

Deployment of the broker as an app running on CloudFoundry is controlled by the ```manifest.yml``` file, which requires no edits. To deploy simply
run ```./deploy.sh cf ceph-objectstore-broker```, with the second argument being the name of the app on CF.

Once the broker is running on CF, it needs to be registered with CF and then the plans need to be made public. To register the broker
use ```cf create-service-broker SERVICE_BROKER BROKER_USERNAME BROKER_PASSWORD BROKER_URL```. Then to make the service public
run ```cf enable-service-access ceph-object-store```, where 'ceph-object-store' is the name of the service provided in ```brokerConfig/service-config.json```.

### Kubernetes & OpenShift

Deployment to k8s and OS are both done by using the following files:

* configMap.yml (Automatically created using your ```vars-file.yml```)
* deployment.yml
* service.yml
* route.yml (only for OS)

To deploy use ```./deploy.sh k8s``` or ```./deploy.sh os```. These commands will set the configMap, deploy the broker and then create a service for it that uses a node port.
In the case of OS, it also creates a route for the broker using the default host.

### Bosh Release

## Integration Tests

To run the tests:
1) Fulfill the required [prerequisites](../README.md#Prerequisites)
2) Run ```./create-configMap```
3) Run ````source tests/tests.env````
4) Run ````go run main.go````
5) In the ```tests``` folder run ````go test```` or ````go test -v```` for more details