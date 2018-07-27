package broker

import (
	"context"
	"errors"

	"github.com/pivotal-cf/brokerapi"
)

type Broker struct {
	ProvisionDetails   brokerapi.ProvisionDetails
	UpdateDetails      brokerapi.UpdateDetails
	DeprovisionDetails brokerapi.DeprovisionDetails

	ProvisionedInstanceIDs   []string
	DeprovisionedInstanceIDs []string
	UpdatedInstanceIDs       []string

	BoundInstanceIDs    []string
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
}
type AsyncServiceBroker struct {
	Broker
	ShouldProvisionAsync bool
}

type AsyncOnlyServiceBroker struct {
	Broker
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
			Name:          "Swift",
			Description:   "Swift service testing something",
			Bindable:      true,
			PlanUpdatable: true,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          broker.PlanID,
					Name:        "Standard",
					Description: "The default plan",
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "Plan for average user",
						Bullets:     []string{},
						Costs: []brokerapi.ServicePlanCost{
							{Unit: "All", Amount: map[string]float64{"CHF": 0}},
						},
					},
					Schemas: &brokerapi.ServiceSchemas{
						Instance: brokerapi.ServiceInstanceSchema{
							Create: brokerapi.Schema{
								Parameters: map[string]interface{}{
									"$schema": "http://json-schema.org/draft-04/schema#",
									"type":    "object",
									"properties": map[string]interface{}{
										"billing-account": map[string]interface{}{
											"description": "Billing account number used to charge use of shared fake server.",
											"type":        "string",
										},
									},
								},
							},
							Update: brokerapi.Schema{
								Parameters: map[string]interface{}{
									"$schema": "http://json-schema.org/draft-04/schema#",
									"type":    "object",
									"properties": map[string]interface{}{
										"billing-account": map[string]interface{}{
											"description": "Billing account number used to charge use of shared fake server.",
											"type":        "string",
										},
									},
								},
							},
						},
						Binding: brokerapi.ServiceBindingSchema{
							Create: brokerapi.Schema{
								Parameters: map[string]interface{}{
									"$schema": "http://json-schema.org/draft-04/schema#",
									"type":    "object",
									"properties": map[string]interface{}{
										"billing-account": map[string]interface{}{
											"description": "Billing account number used to charge use of shared fake server.",
											"type":        "string",
										},
									},
								},
							},
						},
					},
				},
			},
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName:         "Swift",
				ProviderDisplayName: "ZHAW",
				LongDescription:     "Some long description",
				DocumentationUrl:    "http://thedocs.com",
				SupportUrl:          "http://helpme.no",
			},
			Tags: []string{"ZHAW", "Swift"},
		},
	}, nil
}

func (broker *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	broker.BrokerCalled = true

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.ProvisionError != nil {
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if len(broker.ProvisionedInstanceIDs) >= broker.InstanceLimit {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	broker.ProvisionDetails = details
	broker.ProvisionedInstanceIDs = append(broker.ProvisionedInstanceIDs, instanceID)
	return brokerapi.ProvisionedServiceSpec{DashboardURL: broker.DashboardURL}, nil
}

func (broker *AsyncServiceBroker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.ProvisionError != nil {
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if len(broker.ProvisionedInstanceIDs) >= broker.InstanceLimit {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	broker.ProvisionDetails = details
	broker.ProvisionedInstanceIDs = append(broker.ProvisionedInstanceIDs, instanceID)
	return brokerapi.ProvisionedServiceSpec{IsAsync: broker.ShouldProvisionAsync, DashboardURL: broker.DashboardURL, OperationData: broker.OperationDataToReturn}, nil
}

func (broker *AsyncOnlyServiceBroker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.ProvisionError != nil {
		return brokerapi.ProvisionedServiceSpec{}, broker.ProvisionError
	}

	if len(broker.ProvisionedInstanceIDs) >= broker.InstanceLimit {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	broker.ProvisionDetails = details
	broker.ProvisionedInstanceIDs = append(broker.ProvisionedInstanceIDs, instanceID)
	return brokerapi.ProvisionedServiceSpec{IsAsync: true, DashboardURL: broker.DashboardURL}, nil
}

func (broker *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	broker.BrokerCalled = true

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.UpdateError != nil {
		return brokerapi.UpdateServiceSpec{}, broker.UpdateError
	}

	broker.UpdateDetails = details
	broker.UpdatedInstanceIDs = append(broker.UpdatedInstanceIDs, instanceID)
	broker.AsyncAllowed = asyncAllowed
	return brokerapi.UpdateServiceSpec{IsAsync: broker.ShouldReturnAsync, OperationData: broker.OperationDataToReturn}, nil
}

func (broker *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	broker.BrokerCalled = true

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.DeprovisionError != nil {
		return brokerapi.DeprovisionServiceSpec{}, broker.DeprovisionError
	}

	broker.DeprovisionDetails = details
	broker.DeprovisionedInstanceIDs = append(broker.DeprovisionedInstanceIDs, instanceID)

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.DeprovisionServiceSpec{}, nil
	}
	return brokerapi.DeprovisionServiceSpec{IsAsync: false}, brokerapi.ErrInstanceDoesNotExist
}

func (broker *AsyncOnlyServiceBroker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.DeprovisionError != nil {
		return brokerapi.DeprovisionServiceSpec{IsAsync: true}, broker.DeprovisionError
	}

	if !asyncAllowed {
		return brokerapi.DeprovisionServiceSpec{IsAsync: true}, brokerapi.ErrAsyncRequired
	}

	broker.DeprovisionedInstanceIDs = append(broker.DeprovisionedInstanceIDs, instanceID)
	broker.DeprovisionDetails = details

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.DeprovisionServiceSpec{IsAsync: true, OperationData: broker.OperationDataToReturn}, nil
	}

	return brokerapi.DeprovisionServiceSpec{IsAsync: true, OperationData: broker.OperationDataToReturn}, brokerapi.ErrInstanceDoesNotExist
}

func (broker *AsyncServiceBroker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	broker.BrokerCalled = true

	if broker.DeprovisionError != nil {
		return brokerapi.DeprovisionServiceSpec{IsAsync: asyncAllowed}, broker.DeprovisionError
	}

	broker.DeprovisionedInstanceIDs = append(broker.DeprovisionedInstanceIDs, instanceID)
	broker.DeprovisionDetails = details

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		return brokerapi.DeprovisionServiceSpec{IsAsync: asyncAllowed, OperationData: broker.OperationDataToReturn}, nil
	}

	return brokerapi.DeprovisionServiceSpec{OperationData: broker.OperationDataToReturn, IsAsync: asyncAllowed}, brokerapi.ErrInstanceDoesNotExist
}

func (broker *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	broker.BrokerCalled = true

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.BindError != nil {
		return brokerapi.Binding{}, broker.BindError
	}

	broker.BoundBindingDetails = details

	broker.BoundInstanceIDs = append(broker.BoundInstanceIDs, instanceID)
	broker.BoundBindingIDs = append(broker.BoundBindingIDs, bindingID)

	return brokerapi.Binding{
		Credentials: Credentials{
			Host:     "127.0.0.1",
			Port:     3000,
			Username: "batman",
			Password: "robin",
		},
		SyslogDrainURL:  broker.SyslogDrainURL,
		RouteServiceURL: broker.RouteServiceURL,
		VolumeMounts:    broker.VolumeMounts,
	}, nil
}

func (broker *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	broker.BrokerCalled = true

	if val, ok := context.Value("test_context").(bool); ok {
		broker.ReceivedContext = val
	}

	if broker.UnbindError != nil {
		return broker.UnbindError
	}

	broker.UnbindingDetails = details

	if sliceContains(instanceID, broker.ProvisionedInstanceIDs) {
		if sliceContains(bindingID, broker.BoundBindingIDs) {
			return nil
		}
		return brokerapi.ErrBindingDoesNotExist
	}

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

type Credentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func sliceContains(needle string, haystack []string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}
