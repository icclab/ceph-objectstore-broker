package tests

import (
	"encoding/json"
	"github.com/go-resty/resty"
	"github.com/icclab/ceph-objectstore-broker/broker"
	"github.com/icclab/ceph-objectstore-broker/brokerConfig"
	rgw "github.com/icclab/ceph-objectstore-broker/radosgw"
	. "github.com/icclab/ceph-objectstore-broker/tests/testutils"
	"github.com/icclab/ceph-objectstore-broker/utils"
	s3 "github.com/minio/minio-go"
	"github.com/ncw/swift"
	"github.com/pivotal-cf/brokerapi"
	"strconv"
	"strings"
	"testing"
)

type catalog struct {
	Services []brokerapi.Service `json:"services"`
}

type provisionBody struct {
	ServiceID  string `json:"service_id"`
	PlanID     string `json:"plan_id"`
	OrgGUID    string `json:"organization_guid"`
	Space_guid string `json:"space_guid"`
}

type receivedBindCreds struct {
	C broker.BindCreds `json:"credentials"`
}

func TestBroker(t *testing.T) {
	//Load config
	bc := brokerConfig.BrokerConfig{}
	if err := bc.Update(); err != nil {
		t.Fatal("Failed to load broker config")
	}

	s := []brokerapi.Service{}
	if err := utils.LoadJsonFromFile("../brokerConfig/service-config.json", &s); err != nil {
		t.Fatal("Failed to load service config")
	}

	//Catalog
	baseUrl := "http://" + bc.BrokerUsername + ":" + bc.BrokerPassword + "@127.0.0.1:8080/v2"
	resp, err := resty.R().
		SetHeader("X-Broker-API-Version", "2.14").
		Get(baseUrl + "/catalog")

	cat := catalog{}
	unmarshalErr := json.Unmarshal(resp.Body(), &cat)
	if !t.Run("Test Services", CheckErrs(t, nil, err, unmarshalErr, Equals(200, resp.StatusCode(), "Unexpected status code"),
		Equals(len(s), len(cat.Services), "Service count not equal"))) {
		t.FailNow()
	}

	//Provision
	instID := "789"
	provBody := provisionBody{ServiceID: s[0].ID, PlanID: s[0].Plans[0].ID, OrgGUID: "123", Space_guid: "456"}
	req := resty.R().
		SetHeader("X-Broker-API-Version", "2.14").
		SetBody(provBody)

	resp, err = req.Put(baseUrl + "/service_instances/" + instID)
	if !t.Run("Test Provision", CheckErrs(t, nil, err, Equals(201, resp.StatusCode(), "Unexpected status code"))) {
		t.FailNow()
	}

	resp, err = req.Put(baseUrl + "/service_instances/" + instID)
	t.Run("Test Provision Conflict", CheckErrs(t, nil, err, Equals(409, resp.StatusCode(), "Unexpected status code")))

	//Bind
	bindID := "abc"
	resp, err = req.Put(baseUrl + "/service_instances/" + instID + "/service_bindings/" + bindID)
	if !t.Run("Test Bind", CheckErrs(t, nil, err, Equals(201, resp.StatusCode(), "Unexpected status code"))) {
		t.FailNow()
	}

	creds := receivedBindCreds{}
	unmarshalErr = json.Unmarshal(resp.Body(), &creds)
	if unmarshalErr != nil {
		t.Fatal("Failed to parse bind credentials", unmarshalErr)
	}

	resp, err = req.Put(baseUrl + "/service_instances/" + instID + "/service_bindings/" + bindID)
	t.Run("Test Bind Conflict", CheckErrs(t, nil, err, Equals(409, resp.StatusCode(), "Unexpected status code")))

	resp, err = req.Put(baseUrl + "/service_instances/" + instID + "x" + "/service_bindings/" + bindID)
	t.Run("Test Invalid Bind", CheckErrs(t, nil, err, Equals(404, resp.StatusCode(), "Unexpected status code")))

	_, err = s3.New(strings.Replace(creds.C.S3Endpoint, "http://", "", 1), creds.C.S3AccessKey, creds.C.S3SecretKey, false)
	t.Run("Test S3 Creds", CheckErrs(t, nil, err))

	sc := swift.Connection{
		UserName: creds.C.SwiftUser,
		ApiKey:   creds.C.SwiftSecretKey,
		AuthUrl:  creds.C.SwiftEndpoint,
	}
	t.Run("Test Swift Creds", CheckErrs(t, nil, sc.Authenticate()))

	r := rgw.Radosgw{}
	if r.Setup(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosAccessKey, bc.RadosSecretKey) != nil {
		t.Error("Failed to setup radosgw")
	}

	ut := strings.Split(creds.C.S3User, "$")
	q, err := r.GetUserQuotaMB(ut[1], ut[0])
	if err != nil {
		t.Error("Couldn't get user qota from the radosgw", err)
	}
	expectedQ, _ := strconv.Atoi(cat.Services[0].Plans[0].Metadata.AdditionalMetadata["quotaMB"].(string))
	t.Run("Test Initial Plan Size", CheckErrs(t, nil, Equals(expectedQ, q, "Incorrect plan quota")))

	//Update
	provBody.PlanID = s[0].Plans[1].ID
	resp, err = req.SetBody(provBody).Patch(baseUrl + "/service_instances/" + instID)
	t.Run("Test Update", CheckErrs(t, nil, err, Equals(200, resp.StatusCode(), "Unexpected status code")))

	resp, err = req.SetBody(provBody).Patch(baseUrl + "/service_instances/" + instID)
	t.Run("Test Update Conflict", CheckErrs(t, nil, err, Equals(200, resp.StatusCode(), "Unexpected status code")))

	provBody.PlanID = s[0].Plans[1].ID + "x"
	resp, err = req.SetBody(provBody).Patch(baseUrl + "/service_instances/" + instID)
	t.Run("Test Update Invalid Plan", CheckErrs(t, nil, err, Equals(422, resp.StatusCode(), "Unexpected status code")))

	q, _ = r.GetUserQuotaMB(ut[1], ut[0])
	expectedQ, _ = strconv.Atoi(cat.Services[0].Plans[1].Metadata.AdditionalMetadata["quotaMB"].(string))
	t.Run("Test New Plan Size", CheckErrs(t, nil, Equals(expectedQ, q, "Incorrect plan quota")))

	//Unbind
	resp, err = req.
		SetQueryParams(map[string]string{"service_id": s[0].ID, "plan_id": s[0].Plans[1].ID}).
		Delete(baseUrl + "/service_instances/" + instID + "/service_bindings/" + bindID)
	t.Run("Test Unbind", CheckErrs(t, nil, err, Equals(200, resp.StatusCode(), "Unexpected status code")))

	resp, err = req.Delete(baseUrl + "/service_instances/" + instID + "/service_bindings/" + bindID)
	t.Run("Test Unbind Repeat", CheckErrs(t, nil, err, Equals(410, resp.StatusCode(), "Unexpected status code")))

	//Deprovision
	resp, err = req.Delete(baseUrl + "/service_instances/" + instID)
	t.Run("Test Deprovision", CheckErrs(t, nil, err, Equals(200, resp.StatusCode(), "Unexpected status code")))

	resp, err = req.Delete(baseUrl + "/service_instances/" + instID)
	t.Run("Test Deprovision Repeat", CheckErrs(t, nil, err, Equals(410, resp.StatusCode(), "Unexpected status code")))
}
