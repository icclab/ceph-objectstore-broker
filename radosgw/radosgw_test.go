package radosgw

import (
	"github.engineering.zhaw.ch/kaio/swift-go-broker/config"
	"testing"
)

func TestRadosgw(t *testing.T) {
	bc := config.BrokerConfig{}
	if err := config.LoadConfig("../config/broker-config.json", &bc); err != nil {
		t.Fatal("Could not load broker config", err)
	}

	rados := Radosgw{}
	err := rados.Connect(bc.RadosEndpoint, bc.RadosAdminPath, bc.RadosKeyID, bc.RadosSecretKey)
	t.Run("Connect", checkErr(t, err, true, "Failed to connect to the radosgw"))

	user := "test-user"
	subuser := "subuser"
	tenant := "asnfiwejf9w8349t8u023"
	t.Run("CreateUser", checkErr(t, rados.CreateUser(user, user, tenant), true, "Failed to create user", user, tenant))

	_, err = rados.GetUser(user, tenant, false)
	t.Run("GetUser", checkErr(t, err, false, "Failed to get user info"))

	_, err = rados.GetUserUsageMB(user, tenant)
	t.Run("GetUserUsageMB", checkErr(t, err, false, "Failed to get user usage"))

	t.Run("SetUserQuota", checkErr(t, rados.SetUserQuota(user, tenant, 100), false, "Failed to set user quota"))

	_, err = rados.CreateSubuser(user, subuser, tenant)
	t.Run("CreateSubuser", checkErr(t, err, false, "Failed to create subuser", subuser))

	s3Key, err := rados.CreateS3Key(user, tenant)
	t.Run("CreateS3Key", checkErr(t, err, false, "Failed to create S3 Key"))

	t.Run("DeleteS3Key", checkErr(t, rados.DeleteS3Key(user, tenant, s3Key.AccessKey), false, "Failed to delete S3 key", s3Key.AccessKey))

	t.Run("DeleteSubuser", checkErr(t, rados.DeleteSubuser(user, subuser, tenant), false, "Failed to delete subuser"))

	t.Run("DeleteUser", checkErr(t, rados.DeleteUser(user, tenant), false, "Failed to delete user"))
}

func checkErr(t *testing.T, e error, fatal bool, args ...interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		if e != nil {
			if fatal {
				t.Fatal(args, e)
			} else {
				t.Error(args, e)
			}
		}
	}
}
