package main

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	rgw "github.com/myENA/radosgwadmin"
	"github.com/ncw/swift"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/broker"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/config"
	rg "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/radosgw"
	"net/http"
	"os"
)

func main() {
	//Init logger
	logger := lager.NewLogger("Swift-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	logger.Debug("Starting")

	//Load configs
	bc := config.BrokerConfig{}
	err := config.LoadConfig("config/broker-config.json", &bc)
	if err != nil {
		logger.Error("Failed to load broker config", err)
		return
	}

	services := []brokerapi.Service{}
	err = config.LoadConfig("config/service-config.json", &services)
	if err != nil {
		logger.Error("Failed to load service config", err)
		return
	}

	//Connect to rgw
	rados := &rg.Radosgw{}
	if err := rados.Setup(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosKeyID, bc.RadosSecretKey); err != nil {
		logger.Error("Failed to connect to radosgw", err)
		return
	}

	brok := &broker.Broker{
		Logger:        logger,
		Rados:         rados,
		ServiceConfig: services,
		BrokerConfig:  &bc,
		Binds:         make(map[string]broker.Bind),
	}
	creds := brokerapi.BrokerCredentials{Username: bc.BrokerUsername, Password: bc.BrokerPassword}

	//Start the broker
	handler := brokerapi.New(brok, logger, creds)
	http.Handle("/", handler)
	logger.Debug("Handling requests")

	logger.Debug("Listen and serve on port: 8080")
	_ = http.ListenAndServe(":8080", nil)
}

func SwiftFunctionsTests(userInfo *rgw.UserInfoResponse) {
	// Create a connection
	fmt.Println("Swift Tests\n------------")
	c := swift.Connection{
		UserName: "user:subuser",
		ApiKey:   "TmzJur5EULGo0Q1jrfA1b0OzJQprqA3BI2r8zmpT",
		AuthUrl:  "http://160.85.37.79:7480/auth/v1.0",
	}

	// Authenticate
	fmt.Println("Authenticating...")
	err := c.Authenticate()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Authenticated!")
	}

	//Create container
	fmt.Println("\nCreating container...")
	err = c.ContainerCreate("my-container", nil)
	if err != nil {
		fmt.Println("	", err)
	} else {
		fmt.Println("Container created!")
	}

	//Create object
	fmt.Println("\nCreating object...")
	_, err = c.ObjectCreate("my-container", "my-object", false, "", "", nil)
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("Object created!")
	}

	//Put in object
	fmt.Println("\nPutting string in object...")
	err = c.ObjectPutString("my-container", "my-object", "This is in an object :)", "")
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("String placed in object!")
	}

	//Get from object
	fmt.Println("\nReading what was put...")
	content, err := c.ObjectGetString("my-container", "my-object")
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("String received:", "'"+content+"'")
	}

	//Delete object
	fmt.Println("\nDeleting object...")
	err = c.ObjectDelete("my-container", "my-object")
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("Object deleted!")
	}

	//Delete container
	fmt.Println("\nDelete container...")
	err = c.ContainerDelete("my-container")
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("Container deleted!")
	}
}
