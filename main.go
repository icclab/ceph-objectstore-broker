package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/icclab/ceph-objectstore-broker/broker"
	"github.com/icclab/ceph-objectstore-broker/brokerConfig"
	rg "github.com/icclab/ceph-objectstore-broker/radosgw"
	"github.com/icclab/ceph-objectstore-broker/s3"
	"github.com/icclab/ceph-objectstore-broker/utils"
	"github.com/pivotal-cf/brokerapi"
	"net/http"
	"os"
)

func main() {
	//Init logger
	logger := lager.NewLogger("broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	logger.Info("Broker starting")

	//Load configs
	bc := &brokerConfig.BrokerConfig{}
	err := bc.Update()
	if err != nil {
		logger.Error("Failed to load broker config", err)
		return
	}

	services := []brokerapi.Service{}
	err = utils.LoadJsonFromFile("brokerConfig/service-config.json", &services)
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

	//Create s3 client
	s := &s3.S3{}
	err = s.Connect(bc.RadosEndpoint, bc.RadosAccessKey, bc.RadosSecretKey, false)
	if err != nil {
		logger.Error("Failed to connect to S3", err)
		return
	}

	brok := &broker.Broker{
		Logger:            logger,
		Rados:             rados,
		ServiceConfig:     services,
		BrokerConfig:      bc,
		S3:                s,
		ShouldReturnAsync: false,
	}

	if b, _ := s.BucketExists(bc.BucketName); !b {
		if err = s.CreateBucket(bc.BucketName); err != nil {
			logger.Error("Failed to create base bucket for the broker", err)
			return
		}
	}

	//Start the broker
	creds := brokerapi.BrokerCredentials{Username: bc.BrokerUsername, Password: bc.BrokerPassword}
	handler := brokerapi.New(brok, logger, creds)
	http.Handle("/", handler)
	logger.Debug("Listen and serve on port: 8080")
	_ = http.ListenAndServe(":8080", nil)
}
