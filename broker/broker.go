package broker

import (
	"code.cloudfoundry.org/lager"
	"context"
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/swift-go-broker/radosgw"
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

	InstanceLimit int

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

	ServiceID string
	PlanID    string

	Rados   *radosgw.Radosgw
	Logger  lager.Logger
	Secrets map[string]string
	binds   map[string]Bind
}

func (broker *Broker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	broker.BrokerCalled = true

	if val, ok := ctx.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if val, ok := ctx.Value("fails").(bool); ok && val {
		return []brokerapi.Service{}, errors.New("something went wrong!")
	}

	return []brokerapi.Service{
		{
			ID:            broker.ServiceID,
			Name:          "Ceph-Object-Store",
			Description:   "Swift and S3 object store service based on a Ceph backend.",
			Bindable:      true,
			PlanUpdatable: true,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          broker.PlanID,
					Name:        "100MB",
					Description: "100MB object storage",
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "100MB",
					},
				},
			},
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName:         "Ceph Object Store",
				ProviderDisplayName: "ZHAW",
				LongDescription:     "Swift and S3 object store service based on a Ceph backend",
				DocumentationUrl:    "",
				SupportUrl:          "",
			},
			Tags: []string{"Swift", "S3"},
		},
	}, nil
}

func (broker *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	broker.BrokerCalled = true

	//Initial error checking
	if broker.ProvisionError != nil {
		broker.Logger.Error("Provision failed", broker.ProvisionError)
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if len(broker.ProvisionedInstanceIDs) >= broker.InstanceLimit {
		broker.Logger.Error("Provision failed. Max number of instances reached", brokerapi.ErrInstanceLimitMet)
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Provision failed. Instance already exists", brokerapi.ErrInstanceAlreadyExists)
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	//Provision
	if err := broker.Rados.CreateUser(instanceID, instanceID, instanceID); err != nil {
		broker.Logger.Error("Provision failed. Couldn't create user", err)
		return brokerapi.ProvisionedServiceSpec{}, err
	}
	broker.ProvisionDetails = details
	broker.ProvisionedInstanceIDs = append(broker.ProvisionedInstanceIDs, instanceID)
	return brokerapi.ProvisionedServiceSpec{IsAsync: false}, nil
}

func (broker *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.UpdateError != nil {
		return brokerapi.UpdateServiceSpec{}, broker.UpdateError
	}

	broker.UpdateDetails = details
	broker.AsyncAllowed = asyncAllowed
	return brokerapi.UpdateServiceSpec{IsAsync: broker.ShouldReturnAsync, OperationData: broker.OperationDataToReturn}, nil
}

func (broker *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	broker.BrokerCalled = true

	//Error checking
	if broker.DeprovisionError != nil {
		broker.Logger.Error("Deprovision failed", broker.DeprovisionError)
		return brokerapi.DeprovisionServiceSpec{}, broker.DeprovisionError
	}

	if !sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Deprovision failed. Instance does not exist", brokerapi.ErrInstanceDoesNotExist)
		return brokerapi.DeprovisionServiceSpec{IsAsync: false}, brokerapi.ErrInstanceDoesNotExist
	}

	//Deprovision
	if err := broker.Rados.DeleteUser(instanceID, instanceID); err != nil {
		broker.Logger.Error("Deprovision failed. Couldn't delete user", err)
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	removeFromSlice(instanceID, broker.ProvisionedInstanceIDs)
	broker.DeprovisionDetails = details
	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (broker *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	broker.BrokerCalled = true

	if broker.BindError != nil {
		broker.Logger.Error("Bind failed", broker.BindError)
		return brokerapi.Binding{}, broker.BindError
	}

	user := instanceID + "$" + instanceID
	//S3 info
	s3Key, err := broker.Rados.CreateS3Key(user, instanceID)
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't create s3 key", err)
		return brokerapi.Binding{}, err
	}

	//Swift info
	_, err = broker.Rados.CreateSubuser(instanceID, bindingID, instanceID)
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't create swift key (subuser)", err)
		return brokerapi.Binding{}, err
	}

	userInfo, err := broker.Rados.GetUser(instanceID, instanceID)
	if err != nil {
		broker.Logger.Error("Bind failed. Couldn't get user information (to get swift key)", err)
		return brokerapi.Binding{}, err
	}

	//Fill info
	creds := bindCreds{
		S3User:         user,
		S3AccessKey:    s3Key.AccessKey,
		S3SecretKey:    s3Key.SecretKey,
		S3Endpoint:     "http://160.85.37.79:7480",
		SwiftUser:      user + ":" + bindingID,
		SwiftSecretKey: userInfo.SwiftKeys[len(userInfo.SwiftKeys)-1].SecretKey,
		SwiftEndpoint:  "http://160.85.37.79:7480/auth/v1.0",
	}

	//Add to list of binds
	broker.binds[bindingID] = Bind{
		User:        instanceID,
		Subuser:     bindingID,
		Tenant:      instanceID,
		S3AccessKey: creds.S3AccessKey,
		SwiftKey:    creds.SwiftSecretKey,
	}

	broker.BoundBindingDetails = details
	broker.BoundBindingIDs = append(broker.BoundBindingIDs, bindingID)

	return brokerapi.Binding{Credentials: creds}, nil
}

func (broker *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	broker.BrokerCalled = true

	//Error checking
	if broker.UnbindError != nil {
		broker.Logger.Error("Unbind failed", broker.UnbindError)
		return broker.UnbindError
	}

	if !sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		broker.Logger.Error("Unbind failed. Instance not found", brokerapi.ErrInstanceDoesNotExist)
		return brokerapi.ErrInstanceDoesNotExist
	}

	if !sliceContains(bindingID, broker.BoundBindingIDs) {
		broker.Logger.Error("Unbind failed. Binding not found", brokerapi.ErrBindingDoesNotExist)
		return brokerapi.ErrBindingDoesNotExist
	}

	//Delete bind resources
	bind := broker.binds[bindingID]
	if err := broker.Rados.DeleteS3Key(bind.User, bind.Tenant, bind.S3AccessKey); err != nil {
		broker.Logger.Error("Unbind failed. Couldn't delete S3 key", err)
		return err
	}

	if err := broker.Rados.DeleteSubuser(bind.User, bind.Subuser, bind.Tenant); err != nil {
		broker.Logger.Error("Unbind failed. Couldn't delete Swift key (subuser)", err)
		return err
	}
	delete(broker.binds, bindingID)

	broker.UnbindingDetails = details
	return brokerapi.ErrInstanceDoesNotExist
}

func (broker *Broker) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	broker.LastOperationInstanceID = instanceID
	broker.LastOperationData = operationData

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.LastOperationError != nil {
		return brokerapi.LastOperation{}, broker.LastOperationError
	}

	return brokerapi.LastOperation{State: broker.LastOperationState, Description: broker.LastOperationDescription}, nil
}
