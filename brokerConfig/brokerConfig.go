package brokerConfig

import (
	"errors"
	"os"
	"strconv"
)

type BrokerConfig struct {
	RadosAccessKey string
	RadosSecretKey string
	RadosAdminPath string
	RadosEndpoint  string

	S3Endpoint     string
	SwiftEndpoint  string
	BucketName     string
	BrokerUsername string
	BrokerPassword string
	InstanceLimit  int
	InstancePrefix string
}

func (b *BrokerConfig) Update() error {

	const s3Path = "/"
	const swiftPath = "/auth/v1.0"
	const radosAdmin = "admin"
	const instanceLimit = 2000
	const bucketName = "ceph-objectstore-broker"
	const instancePrefix = "instances/"

	//Required params
	if b.RadosAccessKey = os.Getenv("RADOS_ACCESS_KEY"); b.RadosAccessKey == "" {
		return errors.New("RADOS_ACCESS_KEY missing")
	}
	if b.RadosSecretKey = os.Getenv("RADOS_SECRET_KEY"); b.RadosSecretKey == "" {
		return errors.New("RADOS_SECRET_KEY missing")
	}

	if b.RadosEndpoint = os.Getenv("RADOS_ENDPOINT"); b.RadosEndpoint == "" {
		return errors.New("RADOS_ENDPOINT missing")
	}

	if b.BrokerUsername = os.Getenv("BROKER_USERNAME"); b.BrokerUsername == "" {
		return errors.New("BROKER_USERNAME missing")
	}

	if b.BrokerPassword = os.Getenv("BROKER_PASSWORD"); b.BrokerPassword == "" {
		return errors.New("BROKER_PASSWORD missing")
	}

	//Optional params
	if v := os.Getenv("S3_PATH"); v != "" {
		b.S3Endpoint = b.RadosEndpoint + v
	} else {
		b.S3Endpoint = b.RadosEndpoint + s3Path
	}

	if v := os.Getenv("SWIFT_PATH"); v != "" {
		b.SwiftEndpoint = b.RadosEndpoint + v
	} else {
		b.SwiftEndpoint = b.RadosEndpoint + swiftPath
	}

	if v := os.Getenv("BUCKET_NAME"); v != "" {
		b.BucketName = v
	} else {
		b.BucketName = bucketName
	}

	if v := os.Getenv("RADOS_ADMIN"); v != "" {
		b.RadosAdminPath = b.RadosAdminPath + v
	} else {
		b.RadosAdminPath = b.RadosEndpoint + radosAdmin
	}

	if v := os.Getenv("INSTANCE_LIMIT"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil {
			b.InstanceLimit = instanceLimit
			return errors.New("Error reading 'INSTANCE_LIMIT'. Using default value" + strconv.Itoa(instanceLimit))
		}

		b.InstanceLimit = l
	} else {
		b.InstanceLimit = instanceLimit
	}

	if v := os.Getenv("INSTANCE_PREFIX"); v != "" {
		b.InstancePrefix = v
	} else {
		b.InstancePrefix = instancePrefix
	}

	return nil
}
