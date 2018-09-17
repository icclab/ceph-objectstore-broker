# Service Broker for a Ceph Object Store

This broker is compliant with V2 of the [Open Service Broker API](https://www.openservicebrokerapi.org/). The broker provides access to a [Ceph](https://ceph.com/) object store,
and can currently be deployed as CloudFoundry app, on Kubernetes or on OpenShift. Deployment as a Bosh release is planned for the future.

## Table of Contents

* [General Operation](#General-Operation)
* [Deployment](#Deployment)
  * [Prerequisites](#Prerequisites)
  * [CloudFoundry](#CloudFoundry)
  * [Kubernetes & OpenShift](#Kubernetes-&-OpenShift)
  * [Bosh Release](#Bosh-Release)
* [Integration Tests](#Integration-Tests)

<a name="General-Operation"></a>
## General Operation

The service provided by the broker and its plans are in the `brokerConfig/service-config.json` file. You can edit this to your liking before deploying.

When an instance is [provisioned](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#provisioning) a user is created on Ceph. Then when an
application [binds](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#binding) to the broker, it returns access credentials for both the S3 and Swift
APIs supported by Ceph.

The credentials made available to the application (usually through environment variables) after a bind are:-

* s3User
* s3AccessKey
* s3SecretKey
* s3Endpoint
* swiftUser
* swiftSecretKey
* swiftEndpoint

Unbinding and deprovisioning are simply reverse operations of the provision and bind stages.

<a name="Deployment"></a>
## Deployment

Deployment to all platforms is done through the `deploy.sh` file, so once prerequisites for a platform are fulfilled the script can be used to deploy the broker.

<a name="Prerequisites"></a>
### Prerequisites

Before deploying to a platform, you need to provide the required details about your Ceph installation. Specifically you will need a
[Ceph object gateway](http://docs.ceph.com/docs/master/radosgw/) setup. The broker will use the admin user on the gateway to manage users there as required to operate the
service, and so it requires a number of variables including the gateway's endpoint and access keys for the user.

To provide the required information you will need a file called `vars-file.yml`. A template for this file called `vars-file-template.yml` is available, and so can simply
be copied, renamed and then the details filled in.

<a name="CloudFoundry"></a>
### CloudFoundry

Deployment of the broker as an app running on CloudFoundry is controlled by the `manifest.yml` file, which requires no edits. To deploy simply
run `./deploy.sh cf ceph-objectstore-broker`, with the second argument being the name of the app on CF.

Once the broker is running on CF, it needs to be registered with CF and then the plans need to be made public. To register the broker
use `cf create-service-broker SERVICE_BROKER BROKER_USERNAME BROKER_PASSWORD BROKER_URL`. Then to make the service public
run `cf enable-service-access ceph-object-store`, where 'ceph-object-store' is the name of the service provided in `brokerConfig/service-config.json`.

<a name="Kubernetes & OpenShift"></a>
### Kubernetes & OpenShift

Deployment to k8s and OS are both done by using the following files:

* Automatically created/updated using your `vars-file.yml` via the `update-vars` binary (the binary is ran on deploy)
  * config-map.yml
  * secret.yml
* template.yml
* route.yml (only for OS)
* broker.yml (Manually used to register after deployment)

To deploy use `./deploy.sh k8s` or `./deploy.sh os`. These commands will set the config-map, secret, deploy the broker application and then create a service for it. In
the case of OS, it also creates a route for the broker and displays the url of the created route.

The default service created uses a [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) to expose the broker, however depending on your
platform you might want to use something like a loadbalancer, in which case you can just edit the relevant yaml files and then use the deployment script to deploy with your
own configuration.

To register the broker you need to get the url of your broker (it could be deployed on a different platform), any certificates if you want encryption and then update
the `deployment-configs/k8s/broker.yml`. If you don't use encryption then you simply need to set the url field. Once you have update the broker file
you can run `oc apply -f "deployment-configs/k8s/broker.yml"` or `kubectl apply -f "deployment-configs/k8s/broker.yml"`, depending if you are using OpenShift or
Kubernetes.

**NOTE:** To apply the broker you need to be a user with sufficient privileges (e.g. system:admin)

<a name="Bosh-Release"></a>
### Bosh Release

Planned.

<a name="Integration-Tests"></a>
## Integration Tests

To run the tests:
1) Fulfill the required [prerequisites](#Prerequisites)
2) Run `./create-configMap`
3) Run `source tests/tests.env`
4) Run `go run main.go`
5) In the `tests` folder run `go test` or `go test -v` for more details