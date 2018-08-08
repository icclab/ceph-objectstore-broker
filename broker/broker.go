package broker

import (
	"code.cloudfoundry.org/lager"
	"context"
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/config"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/radosgw"
)

type Bind struct {
	S3AccessKey string
	SwiftKey    string
	User        string
	Subuser     string
	Tenant      string
}

type bindCreds struct {
	S3User      string `json:"s3User"`
	S3AccessKey string `json:"s3AcessKey"`
	S3SecretKey string `json:"s3SecretKey"`
	S3Endpoint  string `json:"s3Endpoint"`

	SwiftUser      string `json:"swiftUser"`
	SwiftSecretKey string `json:"swiftSecretKey"`
	SwiftEndpoint  string `json:"swiftEndpoint"`
}

type Broker struct {
	ProvisionDetails   brokerapi.ProvisionDetails
	UpdateDetails      brokerapi.UpdateDetails
	DeprovisionDetails brokerapi.DeprovisionDetails

	ProvisionedInstanceIDs []string

	BoundBindingIDs     []string
	BoundBindingDetails brokerapi.BindDetails
	SyslogDrainURL      string
	RouteServiceURL     string
	VolumeMounts        []brokerapi.VolumeMount

	UnbindingDetails brokerapi.UnbindDetails

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
	DashboardURL          string
	OperationDataToReturn string

	LastOperationInstanceID string
	LastOperationData       string

	ReceivedContext bool

	Rados         *radosgw.Radosgw
	Logger        lager.Logger
	ServiceConfig []brokerapi.Service
	BrokerConfig  *config.BrokerConfig
	Binds         map[string]Bind
}

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
		broker.Logger.Error("Provision failed", broker.ProvisionError)
		broker.LastOperationError = broker.ProvisionError
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if len(broker.ProvisionedInstanceIDs) >= broker.BrokerConfig.InstanceLimit {
		broker.Logger.Error("Provision failed. Max number of instances reached", brokerapi.ErrInstanceLimitMet)
		broker.LastOperationError = brokerapi.ErrInstanceLimitMet
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Provision failed. Instance already exists", brokerapi.ErrInstanceAlreadyExists)
		broker.LastOperationError = brokerapi.ErrInstanceAlreadyExists
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	//Provision
	if err := broker.Rados.CreateUser(instanceID, instanceID, createTenantID(instanceID)); err != nil {
		broker.Logger.Error("Provision failed. Couldn't create user", err)
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	quota, err := getPlanQuota(details.PlanID, broker.ServiceConfig[0].Plans)
	if err != nil {
		broker.Logger.Error("Provision failed. Couldn't get plan size quota", err)
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	if err := broker.Rados.SetUserQuota(instanceID, createTenantID(instanceID), quota); err != nil {
		broker.Logger.Error("Provision failed. Couldn't set user quota", err)
		broker.LastOperationError = err
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	broker.ProvisionDetails = details
	broker.ProvisionedInstanceIDs = append(broker.ProvisionedInstanceIDs, instanceID)
	broker.LastOperationError = nil

	return brokerapi.ProvisionedServiceSpec{IsAsync: false}, nil
}

func (broker *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.UpdateError != nil {
		broker.Logger.Error("Update failed", broker.ProvisionError)
		broker.LastOperationError = broker.UpdateError
		return brokerapi.UpdateServiceSpec{}, broker.UpdateError
	}

	//Update
	newPlanQuota, err := getPlanQuota(details.PlanID, broker.ServiceConfig[0].Plans)
	if err != nil {
		broker.Logger.Error("Update failed. couldn't get new plan quota", err)
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	usage, err := broker.Rados.GetUserUsageMB(instanceID, createTenantID(instanceID))
	if err != nil {
		broker.Logger.Error("Update failed. couldn't get user usage", err)
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	if usage >= newPlanQuota {
		err = errors.New("Current object store usage exceeds size quota of the new plan")
		broker.Logger.Error("Update failed. Must reduce usage first", err)
		broker.LastOperationError = err
		return brokerapi.UpdateServiceSpec{}, err
	}

	if err := broker.Rados.SetUserQuota(instanceID, createTenantID(instanceID), newPlanQuota); err != nil {
		broker.Logger.Error("Update failed. Couldn't update quota for the new plan", err)
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
		broker.Logger.Error("Deprovision failed", broker.DeprovisionError)
		broker.LastOperationError = broker.DeprovisionError
		return brokerapi.DeprovisionServiceSpec{}, broker.DeprovisionError
	}

	if !sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Deprovision failed. Instance does not exist", brokerapi.ErrInstanceDoesNotExist)
		broker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
		return brokerapi.DeprovisionServiceSpec{IsAsync: false}, brokerapi.ErrInstanceDoesNotExist
	}

	//Deprovision
	if err := broker.Rados.DeleteUser(instanceID, createTenantID(instanceID)); err != nil {
		broker.Logger.Error("Deprovision failed. Couldn't delete user", err)
		broker.LastOperationError = err
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	removeFromSlice(instanceID, broker.ProvisionedInstanceIDs)
	broker.DeprovisionDetails = details
	broker.LastOperationError = nil

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (broker *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	broker.BrokerCalled = true

	if broker.BindError != nil {
		broker.Logger.Error("Bind failed", broker.BindError)
		broker.LastOperationError = broker.BindError
		return brokerapi.Binding{}, broker.BindError
	}

	//S3 info
	s3Key, err := broker.Rados.CreateS3Key(instanceID, createTenantID(instanceID))
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't create s3 key", err)
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	//Swift info
	_, err = broker.Rados.CreateSubuser(instanceID, bindingID, createTenantID(instanceID))
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't create swift key (subuser)", err)
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	userInfo, err := broker.Rados.GetUser(instanceID, createTenantID(instanceID), false)
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't get user information (to get swift key)", err)
		broker.LastOperationError = err
		return brokerapi.Binding{}, err
	}

	//Fill info
	user := createTenantID(instanceID) + "$" + instanceID
	creds := bindCreds{
		S3User:         user,
		S3AccessKey:    s3Key.AccessKey,
		S3SecretKey:    s3Key.SecretKey,
		S3Endpoint:     broker.BrokerConfig.S3Endpoint,
		SwiftUser:      user + ":" + bindingID,
		SwiftSecretKey: userInfo.SwiftKeys[len(userInfo.SwiftKeys)-1].SecretKey,
		SwiftEndpoint:  broker.BrokerConfig.SwiftEndpoint,
	}

	//Add to list of binds

	broker.Binds[bindingID] = Bind{
		User:        instanceID,
		Subuser:     bindingID,
		Tenant:      createTenantID(instanceID),
		S3AccessKey: creds.S3AccessKey,
		SwiftKey:    creds.SwiftSecretKey,
	}

	broker.BoundBindingDetails = details
	broker.BoundBindingIDs = append(broker.BoundBindingIDs, bindingID)
	broker.LastOperationError = nil

	return brokerapi.Binding{Credentials: creds}, nil
}

func (broker *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	broker.BrokerCalled = true

	//Error checking
	if broker.UnbindError != nil {
		broker.Logger.Error("Unbind failed", broker.UnbindError)
		broker.LastOperationError = broker.UnbindError
		return broker.UnbindError
	}

	if !sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Unbind failed. Instance not found", brokerapi.ErrInstanceDoesNotExist)
		broker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
		return brokerapi.ErrInstanceDoesNotExist
	}

	if !sliceContains(bindingID, broker.BoundBindingIDs) {
		broker.Logger.Error("Unbind failed. Binding not found", brokerapi.ErrBindingDoesNotExist)
		broker.LastOperationError = brokerapi.ErrBindingDoesNotExist
		return brokerapi.ErrBindingDoesNotExist
	}

	//Delete bind resources
	bind := broker.Binds[bindingID]
	if err := broker.Rados.DeleteS3Key(bind.User, bind.Tenant, bind.S3AccessKey); err != nil {
		broker.Logger.Error("Unbind failed. Couldn't delete S3 key", err)
		broker.LastOperationError = err
		return err
	}

	if err := broker.Rados.DeleteSubuser(bind.User, bind.Subuser, bind.Tenant); err != nil {
		broker.Logger.Error("Unbind failed. Couldn't delete Swift key (subuser)", err)
		broker.LastOperationError = err
		return err
	}
	delete(broker.Binds, bindingID)
	removeFromSlice(bindingID, broker.BoundBindingIDs)

	broker.UnbindingDetails = details
	broker.LastOperationError = nil

	return brokerapi.ErrInstanceDoesNotExist
}

func (broker *Broker) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	broker.LastOperationInstanceID = instanceID
	broker.LastOperationData = operationData

	if broker.LastOperationError != nil {
		return brokerapi.LastOperation{}, broker.LastOperationError
	}

	return brokerapi.LastOperation{State: broker.LastOperationState, Description: broker.LastOperationDescription}, nil
}
