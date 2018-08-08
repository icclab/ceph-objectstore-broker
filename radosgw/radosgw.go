package radosgw

import (
	"context"
	rgw "github.com/myENA/radosgwadmin"
	rcl "github.com/myENA/restclient"
	"time"
)

type Radosgw struct {
	conn      *rgw.AdminAPI
	keyID     string
	secretKey string
}

func (rg *Radosgw) Setup(radosUrl string, radosAdminPath string, keyID string, secretKey string) error {
	rg.keyID = keyID
	rg.secretKey = secretKey

	cfg := &rgw.Config{
		ClientConfig: rcl.ClientConfig{
			ClientTimeout: rcl.Duration(time.Second * 5),
		},
		ServerURL:       radosUrl,
		AdminPath:       radosAdminPath,
		AccessKeyID:     rg.keyID,
		SecretAccessKey: rg.secretKey,
	}

	conn, err := rgw.NewAdminAPI(cfg)
	if err != nil {
		return err
	}

	rg.conn = conn
	return nil
}

func (rg *Radosgw) CreateUser(name string, dispName string, tenant string) error {
	_, err := rg.conn.UserCreate(context.Background(), &rgw.UserCreateRequest{UID: name, DisplayName: dispName, Tenant: tenant})
	if err != nil {
		return err
	}

	return nil
}

func (rg *Radosgw) GetUser(name string, tenant string, getStats bool) (*rgw.UserInfoResponse, error) {
	userInfo, err := rg.conn.UserInfo(context.Background(), tenant+"$"+name, getStats)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (rg *Radosgw) SetUserQuota(name string, tenant string, sizeMB int) error {
	err := rg.conn.QuotaSet(context.Background(), &rgw.QuotaSetRequest{UID: tenant + "$" + name, QuotaType: "user", MaximumSizeKb: sizeMB * 1024, Enabled: true})
	if err != nil {
		return err
	}

	return nil
}

func (rg *Radosgw) GetUserQuotaMB(name string, tenant string) (int, error) {
	q, err := rg.conn.QuotaUser(context.Background(), tenant+"$"+name)
	if err != nil {
		return -1, err
	}

	return (int)(q.MaxSizeKb / 1024), nil
}

func (rg *Radosgw) GetUserUsageMB(name string, tenant string) (int, error) {
	userInfo, err := rg.GetUser(name, tenant, true)
	if err != nil {
		return -1, err
	}

	return userInfo.Stats.SizeKB / 1024, nil
}

func (rg *Radosgw) DeleteUser(name string, tenant string) error {
	err := rg.conn.UserRm(context.Background(), tenant+"$"+name, true)
	if err != nil {
		return err
	}

	return nil
}

//Creating a subuser creates a swift key
func (rg *Radosgw) CreateSubuser(user string, subuser string, tenant string) (*rgw.SubUser, error) {
	subusers, err := rg.conn.SubUserCreate(context.Background(), &rgw.SubUserCreateModifyRequest{UID: tenant + "$" + user, SubUser: subuser, Access: "readwrite"})
	if err != nil {
		return nil, err
	}

	return &subusers[len(subusers)-1], nil
}

func (rg *Radosgw) DeleteSubuser(user string, subuser string, tenant string) error {
	purge := true
	err := rg.conn.SubUserRm(context.Background(), &rgw.SubUserRmRequest{UID: tenant + "$" + user, SubUser: subuser, PurgeKeys: &purge})
	if err != nil {
		return err
	}

	return nil
}

func (rg *Radosgw) CreateS3Key(user string, tenant string) (*rgw.UserKey, error) {
	genKey := true

	keys, err := rg.conn.KeyCreate(context.Background(), &rgw.KeyCreateRequest{UID: tenant + "$" + user, GenerateKey: &genKey})
	if err != nil {
		return nil, err
	}

	return &keys[len(keys)-1], nil
}

func (rg *Radosgw) DeleteS3Key(user string, tenant string, s3AccessKey string) error {
	err := rg.conn.KeyRm(context.Background(), &rgw.KeyRmRequest{UID: tenant + "$" + user, AccessKey: s3AccessKey})
	if err != nil {
		return err
	}

	return nil
}
