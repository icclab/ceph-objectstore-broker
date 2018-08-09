package tests

import (
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/brokerConfig"
	rgw "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/radosgw"
	. "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/testutils"
	"testing"
)

func TestRadosgw(t *testing.T) {
	bc := brokerConfig.BrokerConfig{}
	if err := bc.Update(); err != nil {
		t.Fatal("Could not load broker config", err)
	}

	rados := rgw.Radosgw{}
	err := rados.Setup(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosAccessKey, bc.RadosSecretKey)
	if !t.Run("Connect", CheckErrs(t, nil, err)) {
		t.FailNow()
	}

	//Vars to use
	user := "test-user"
	subuser := "subuser"
	tenant := "asnfiwejf9w8349t8u023"
	quotaSize := 100
	if !t.Run("Create User", CheckErrs(t, nil, rados.CreateUser(user, user, tenant))) {
		t.FailNow()
	}

	userInfo, err := rados.GetUser(user, tenant, false)
	t.Run("Get User", CheckErrs(t, nil, err, Equals(user, userInfo.UserID, "Returned uid incorrect"), Equals(tenant, userInfo.Tenant, "Returned tenant incorrect")))

	usage, err := rados.GetUserUsageMB(user, tenant)
	t.Run("Get User UsageMB", CheckErrs(t, nil, err, Equals(0, usage, "User usage is incorrect")))

	t.Run("Set User Quota", CheckErrs(t, nil, rados.SetUserQuota(user, tenant, quotaSize)))

	q, err := rados.GetUserQuotaMB(user, tenant)
	t.Run("Get User QuotaMB", CheckErrs(t, nil, err, Equals(quotaSize, q, "Returned quota size is incorrect")))

	subuserInfo, err := rados.CreateSubuser(user, subuser, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("Create Subuser", CheckErrs(t, nil, err, Equals(tenant+"$"+user+":"+subuser, subuserInfo.ID, "Returned subuser is incorrect"),
		Equals(1, len(userInfo.SubUsers), "Wrong number of subusers")))

	s3Key, err := rados.CreateS3Key(user, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("Create S3 Key", CheckErrs(t, nil, err, Equals(2, len(userInfo.Keys), "Wrong number of keys")))

	err = rados.DeleteS3Key(user, tenant, s3Key.AccessKey)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("Delete S3 Key", CheckErrs(t, nil, err, Equals(1, len(userInfo.Keys), "Wrong number of keys")))

	err = rados.DeleteSubuser(user, subuser, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("Delete Subuser", CheckErrs(t, nil, err, Equals(0, len(userInfo.SubUsers), "Wrong number of subusers")))

	t.Run("Delete User", CheckErrs(t, nil, rados.DeleteUser(user, tenant)))
}
