package radosgw

import (
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/config"
	. "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/testutils"
	"testing"
)

func TestRadosgw(t *testing.T) {
	bc := config.BrokerConfig{}
	if err := config.LoadConfig("../config/broker-config.json", &bc); err != nil {
		t.Fatal("Could not load broker config", err)
	}

	rados := Radosgw{}
	err := rados.Setup(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosKeyID, bc.RadosSecretKey)
	t.Run("Connect", CheckErrs(t, nil, true, err))

	//Vars to use
	user := "test-user"
	subuser := "subuser"
	tenant := "asnfiwejf9w8349t8u023"
	quotaSize := 100
	t.Run("CreateUser", CheckErrs(t, nil, true, rados.CreateUser(user, user, tenant)))

	userInfo, err := rados.GetUser(user, tenant, false)
	t.Run("GetUser", CheckErrs(t, nil, false, err, Equals(user, userInfo.UserID, "Returned uid incorrect"), Equals(tenant, userInfo.Tenant, "Returned tenant incorrect")))

	usage, err := rados.GetUserUsageMB(user, tenant)
	t.Run("GetUserUsageMB", CheckErrs(t, nil, false, err, Equals(0, usage, "User usage is incorrect")))

	t.Run("SetUserQuota", CheckErrs(t, nil, false, rados.SetUserQuota(user, tenant, quotaSize)))

	q, err := rados.GetUserQuotaMB(user, tenant)
	t.Run("GetUserQuotaMB", CheckErrs(t, nil, false, err, Equals(quotaSize, q, "Returned quota size is incorrect")))

	subuserInfo, err := rados.CreateSubuser(user, subuser, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("CreateSubuser", CheckErrs(t, nil, false, err, Equals(tenant+"$"+user+":"+subuser, subuserInfo.ID, "Returned subuser is incorrect"),
		Equals(1, len(userInfo.SubUsers), "Wrong number of subusers")))

	s3Key, err := rados.CreateS3Key(user, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("CreateS3Key", CheckErrs(t, nil, false, err, Equals(2, len(userInfo.Keys), "Wrong number of keys")))

	err = rados.DeleteS3Key(user, tenant, s3Key.AccessKey)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("DeleteS3Key", CheckErrs(t, nil, false, err, Equals(1, len(userInfo.Keys), "Wrong number of keys")))

	err = rados.DeleteSubuser(user, subuser, tenant)
	userInfo, _ = rados.GetUser(user, tenant, false)
	t.Run("DeleteSubuser", CheckErrs(t, nil, false, err, Equals(0, len(userInfo.SubUsers), "Wrong number of subusers")))

	t.Run("DeleteUser", CheckErrs(t, nil, false, rados.DeleteUser(user, tenant)))
}
