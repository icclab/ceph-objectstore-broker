package broker

import (
	"code.cloudfoundry.org/lager"
	"context"
	"encoding/json"
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/brokerConfig"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/radosgw"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/s3"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/utils"
)

type Bind struct {
	S3AccessKey string `json:"s3AccessKey"`
	SwiftKey    string `json:"swiftKey"`
	User        string `json:"user"`
	Subuser     string `json:"subuser"`
	Tenant      string `json:"tenant"`
}

type BindCreds struct {
	S3User      string `json:"s3User"`
	S3AccessKey string `json:"s3AcessKey"`
	S3SecretKey string `json:"s3SecretKey"`
	S3Endpoint  string `json:"s3Endpoint"`

	SwiftUser      string `json:"swiftUser"`
	SwiftSecretKey string `json:"swiftSecretKey"`
	SwiftEndpoint  string `json:"swiftEndpoint"`
}

type Broker struct {
	ProvisionDetails    brokerapi.ProvisionDetails
	UpdateDetails       brokerapi.UpdateDetails
	DeprovisionDetails  brokerapi.DeprovisionDetails
	BoundBindingDetails brokerapi.BindDetails
	UnbindingDetails    brokerapi.UnbindDetails

	ProvisionError     error
	BindError          error
	UnbindError        error
	DeprovisionError   error
	LastOperationError error
	UpdateError        error

	BrokerCalled             bool
	LastOperationState       brokerapi.LastOperationState
	LastOperationDescription string

	AsyncAllowed bool

	ShouldReturnAsync     bool
	OperationDataToReturn string

	LastOperationInstanceID string
	LastOperationData       string

	Rados         *radosgw.Radosgw
	Logger        lager.Logger
	ServiceConfig []brokerapi.Service
	BrokerConfig  *brokerConfig.BrokerConfig
	//Maps a bindID to a bind struct
	S3 *s3.S3
}

const BucketName = "ceph-objectstore-broker"
const instancePrefix = "instances/"

func (broker *Broker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	broker.BrokerCalled = true
	//All possible service-config can be found here: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#input-parameters-schema-object
	broker.LastOperationError = nil
	return broker.ServiceConfig, nil
}

func (broker *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	broker.BrokerCalled = true

	//Initial error checking
	if broker.ProvisionError != nil {
		broker.LastOperationError = broker.ProvisionError
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if broker.provisionCount() >= broker.BrokerConfig.InstanceLimit {
		broker.LastOperationError = brokerapi.ErrInstanceLimitMet
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if broker.instanceExists(instanceID) {
		broker.LastOperationError = brokerapi.ErrInstanceAlreadyExists
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	//Provision
	if err := broker.Rados.CreateUser(instanceID, instanceID, createTenantID(instanceID)); err != nil {
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	quota, err := broker.getPlanQuota(details.PlanID)
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	if err := broker.Rados.SetUserQuota(instanceID, createTenantID(instanceID), quota); err != nil {
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	if err := broker.S3.PutObject(BucketName, getInstanceObjName(instanceID), ""); err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	broker.ProvisionDetails = details
	broker.AsyncAllowed = asyncAllowed
	broker.LastOperationError = nil

	return brokerapi.ProvisionedServiceSpec{IsAsync: false}, nil
}

func (broker *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.UpdateError != nil {
		broker.LastOperationError = broker.UpdateError
		return brokerapi.UpdateServiceSpec{}, broker.UpdateError
	}

	if _, err := broker.getPlan(details.PlanID); err != nil {
		broker.LastOperationError = brokerapi.ErrPlanChangeNotSupported
		return brokerapi.UpdateServiceSpec{}, brokerapi.ErrPlanChangeNotSupported
	}

	if details.PlanID == details.PreviousValues.PlanID {
		broker.LastOperationError = nil
		return brokerapi.UpdateServiceSpec{}, nil
	}

	//Update
	newPlanQuota, err := broker.getPlanQuota(details.PlanID)
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	usage, err := broker.Rados.GetUserUsageMB(instanceID, createTenantID(instanceID))
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	if usage >= newPlanQuota {
		err = errors.New("Current object store usage exceeds size quota of the new plan")
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	if err := broker.Rados.SetUserQuota(instanceID, createTenantID(instanceID), newPlanQuota); err != nil {
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	broker.UpdateDetails = details
	broker.AsyncAllowed = asyncAllowed
	broker.LastOperationError = nil

	return brokerapi.UpdateServiceSpec{IsAsync: broker.ShouldReturnAsync, OperationData: broker.OperationDataToReturn}, nil
}

func (broker *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	broker.BrokerCalled = true

	//Error checking
	if broker.DeprovisionError != nil {
		broker.LastOperationError = broker.DeprovisionError
		return brokerapi.DeprovisionServiceSpec{}, broker.DeprovisionError
	}

	if !broker.instanceExists(instanceID) {
		broker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
		return brokerapi.DeprovisionServiceSpec{IsAsync: false}, brokerapi.ErrInstanceDoesNotExist
	}

	if broker.hasBinds(instanceID) {
		err := brokerapi.NewFailureResponse(errors.New("Deprovision failed because the instance has binds. All binds under this instance must be unbound before deprovisioning."),
			403, "deprovision-with-existing-binds")
		broker.LastOperationError = err
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	//Deprovision
	if err := broker.Rados.DeleteUser(instanceID, createTenantID(instanceID)); err != nil {
		broker.LastOperationError = err
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	if err := broker.S3.DeleteObject(BucketName, getInstanceObjName(instanceID)); err != nil {
		broker.LastOperationError = err
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	broker.DeprovisionDetails = details
	broker.AsyncAllowed = asyncAllowed
	broker.LastOperationError = nil

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (broker *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	broker.BrokerCalled = true

	if broker.BindError != nil {
		broker.LastOperationError = broker.BindError
		return brokerapi.Binding{}, broker.BindError
	}

	if !broker.instanceExists(instanceID) {
		broker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
		return brokerapi.Binding{}, brokerapi.ErrInstanceDoesNotExist
	}

	if broker.bindingExists(instanceID, bindingID) {
		broker.LastOperationError = brokerapi.ErrBindingAlreadyExists
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}

	//S3 info
	s3Key, err := broker.Rados.CreateS3Key(instanceID, createTenantID(instanceID))
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	//Swift info
	_, err = broker.Rados.CreateSubuser(instanceID, bindingID, createTenantID(instanceID))
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	userInfo, err := broker.Rados.GetUser(instanceID, createTenantID(instanceID), false)
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	//Fill info
	user := createTenantID(instanceID) + "$" + instanceID
	creds := BindCreds{
		S3User:         user,
		S3AccessKey:    s3Key.AccessKey,
		S3SecretKey:    s3Key.SecretKey,
		S3Endpoint:     broker.BrokerConfig.S3Endpoint,
		SwiftUser:      user + ":" + bindingID,
		SwiftSecretKey: userInfo.SwiftKeys[len(userInfo.SwiftKeys)-1].SecretKey,
		SwiftEndpoint:  broker.BrokerConfig.SwiftEndpoint,
	}

	//Store bind information
	b := Bind{
		User:        instanceID,
		Subuser:     bindingID,
		Tenant:      createTenantID(instanceID),
		S3AccessKey: creds.S3AccessKey,
		SwiftKey:    creds.SwiftSecretKey,
	}

	j, err := json.Marshal(b)
	if err != nil {
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	if err := broker.S3.PutObject(BucketName, getBindObjName(instanceID, bindingID), string(j)); err != nil {
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	broker.BoundBindingDetails = details
	broker.LastOperationError = nil

	return brokerapi.Binding{Credentials: creds}, nil
}

func (broker *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	broker.BrokerCalled = true

	//Error checking
	if broker.UnbindError != nil {
		broker.LastOperationError = broker.UnbindError
		return broker.UnbindError
	}

	if !broker.instanceExists(instanceID) {
		broker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
		return brokerapi.ErrInstanceDoesNotExist
	}

	if !broker.bindingExists(instanceID, bindingID) {
		broker.LastOperationError = brokerapi.ErrBindingDoesNotExist
		return brokerapi.ErrBindingDoesNotExist
	}

	//Delete bind resources
	j, err := broker.S3.GetObjectString(BucketName, getBindObjName(instanceID, bindingID))
	if err != nil {
		broker.LastOperationError = err
		return err
	}

	bind := Bind{}
	err = utils.LoadJson(j, &bind)
	if err != nil {
		broker.LastOperationError = err
		return err
	}

	if err := broker.Rados.DeleteS3Key(bind.User, bind.Tenant, bind.S3AccessKey); err != nil {
		broker.LastOperationError = err
		return err
	}

	if err := broker.Rados.DeleteSubuser(bind.User, bind.Subuser, bind.Tenant); err != nil {
		broker.LastOperationError = err
		return err
	}

	if err := broker.S3.DeleteObject(BucketName, getBindObjName(instanceID, bindingID)); err != nil {
		broker.LastOperationError = err
		return err
	}

	broker.UnbindingDetails = details
	broker.LastOperationError = nil

	return nil
}

func (broker *Broker) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	broker.LastOperationInstanceID = instanceID
	broker.LastOperationData = operationData

	if broker.LastOperationError != nil {
		return brokerapi.LastOperation{}, broker.LastOperationError
	}

	return brokerapi.LastOperation{State: broker.LastOperationState, Description: broker.LastOperationDescription}, nil
}
