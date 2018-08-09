package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/broker"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/brokerConfig"
	rg "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/radosgw"
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/utils"
	"net/http"
	"os"
)

func main() {
	//Init logger
	logger := lager.NewLogger("broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	logger.Debug("Starting")

	//Load configs
	bc := &brokerConfig.BrokerConfig{}
	err := bc.Update()
	if err != nil {
		logger.Error("Failed to load broker config", err)
		return
	}

	services := []brokerapi.Service{}
	err = utils.LoadJson("brokerConfig/service-config.json", &services)
	if err != nil {
		logger.Error("Failed to load service config", err)
		return
	}

	//Connect to rgw
	rados := &rg.Radosgw{}
	if err := rados.Setup(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosAccessKey, bc.RadosSecretKey); err != nil {
		logger.Error("Failed to connect to radosgw", err)
		return
	}

	brok := &broker.Broker{
		Logger:        logger,
		Rados:         rados,
		ServiceConfig: services,
		BrokerConfig:  bc,
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
