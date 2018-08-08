package radosgw

import (
	"github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/config"
	. "github.engineering.zhaw.ch/kaio/ceph-objectstore-broker/testutils"
	"strconv"
	"testing"
)

func TestRadosgw(t *testing.T) {
	bc := config.BrokerConfig{}
	if err := config.LoadConfig("../config/broker-config.json", &bc); err != nil {
		t.Fatal("Could not load broker config", err)
	}

	rados := Radosgw{}
	err := rados.Connect(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosKeyID, bc.RadosSecretKey)
	t.Run("Connect", CheckErr(t, err, true, "Failed to connect to the radosgw"))

	user := "test-user"
	subuser := "subuser"
	tenant := "asnfiwejf9w8349t8u023"
	t.Run("CreateUser", CheckErr(t, rados.CreateUser(user, user, tenant), true, "Failed to create user", user, tenant))

	userInfo, err := rados.GetUser(user, tenant, false)
	t.Run("GetUser", CheckErrs(t, []error{err, Check(userInfo.UserID == user, "Returned uid '"+userInfo.UserID+"' incorrect"),
		Check(userInfo.Tenant == tenant, "Returned tenant '"+userInfo.Tenant+"' incorrect")},
		false, "Failed to get user info"))

	usage, err := rados.GetUserUsageMB(user, tenant)
	t.Run("GetUserUsageMB", CheckErrs(t, []error{err, Check(usage == 0, "User usage '"+strconv.Itoa(usage)+"' is incorrect")}, false, "Failed to get user usage"))

	t.Run("SetUserQuota", CheckErr(t, rados.SetUserQuota(user, tenant, 100), false, "Failed to set user quota"))

	q, err := rados.GetUserQuotaMB(user, tenant)
	t.Run("GetUserQuotaMB", CheckErrs(t, []error{err, Check(q == 100, "Returned quota size '"+strconv.Itoa(q)+"' is incorrect")},
		false, "Failed to get user quota"))

	subuserInfo, err := rados.CreateSubuser(user, subuser, tenant)
	t.Run("CreateSubuser", CheckErrs(t, []error{err, Check(subuserInfo.ID == tenant+"$"+user+":"+subuser, "Returned subuser '"+subuserInfo.ID+"' is incorrect")},
		false, "Failed to create subuser", subuser))

	s3Key, err := rados.CreateS3Key(user, tenant)
	t.Run("CreateS3Key", CheckErr(t, err, false, "Failed to create S3 Key"))

	t.Run("DeleteS3Key", CheckErr(t, rados.DeleteS3Key(user, tenant, s3Key.AccessKey), false, "Failed to delete S3 key", s3Key.AccessKey))

	t.Run("DeleteSubuser", CheckErr(t, rados.DeleteSubuser(user, subuser, tenant), false, "Failed to delete subuser"))

	t.Run("DeleteUser", CheckErr(t, rados.DeleteUser(user, tenant), false, "Failed to delete user"))
}
