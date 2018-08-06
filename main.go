package main

import (
	"bufio"
	"code.cloudfoundry.org/lager"
	"context"
	"fmt"
	rgw "github.com/myENA/radosgwadmin"
	rcl "github.com/myENA/restclient"
	"github.com/ncw/swift"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/swift-go-broker/broker"
	rg "github.engineering.zhaw.ch/kaio/swift-go-broker/radosgw"
	"net/http"
	"os"
	"time"
)

const (
	radosAdminPath = "cf-go-broker"
	radosUrl       = "http://160.85.37.79:7480"
	swiftUrl       = "http://160.85.37.79:7480/auth/v1.0"
)

func main() {
	//Init logger
	logger := lager.NewLogger("Swift-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	logger.Debug("Starting")

	//Init broker
	secrets := LoadSecrets()
	rados := &rg.Radosgw{}
	if err := rados.Connect(radosUrl, radosAdminPath, secrets["keyID"], secrets["secretKey"]); err != nil {
		logger.Error("Failed to connect to radosgw", err)
		return
	}

	broker := &broker.Broker{
		ServiceID: "1234-4321-abcd-fghi",
		PlanID:    "4321-1234-abcd-fghi",
		Secrets:   secrets,
		Logger:    logger,
		Rados:     rados,
	}
	creds := brokerapi.BrokerCredentials{Username: secrets["adminUsername"], Password: secrets["adminPassword"]}

	//Start the broker
	handler := brokerapi.New(broker, logger, creds)
	http.Handle("/", handler)
	logger.Debug("Handling requests")

	logger.Debug("Listen and serve on port: 8080")
	// _ = http.ListenAndServe(":8080", nil)

	StartTests(secrets)
}

func LoadSecrets() map[string]string {
	secrets := map[string]string{}

	f, _ := os.Open("broker-secrets")
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		key := scanner.Text()
		scanner.Scan()
		secrets[key] = scanner.Text()
	}

	return secrets
}

func StartTests(secrets map[string]string) {
	userInfo := UserCreationTests(secrets)
	// time.Sleep(5 * time.Second)
	SwiftFunctionsTests(userInfo)
	UserDeletionTests(userInfo, secrets)
}

func UserCreationTests(secrets map[string]string) *rgw.UserInfoResponse {
	//Initial connection
	fmt.Println("\nUser Creation Tests\n------------")
	cfg := &rgw.Config{
		ClientConfig: rcl.ClientConfig{
			ClientTimeout: rcl.Duration(time.Second * 10),
		},
		ServerURL:       radosUrl,
		AdminPath:       radosAdminPath,
		AccessKeyID:     secrets["keyID"],
		SecretAccessKey: secrets["secretKey"],
	}

	aa, err := rgw.NewAdminAPI(cfg)
	if err != nil {
		fmt.Println("Admin API error\n\n", err)
	}

	//Create user
	fmt.Println("Creating new user...")
	userInfo, err := aa.UserCreate(context.Background(), &rgw.UserCreateRequest{UID: "new-user", DisplayName: "my-new-user", Tenant: "a"})
	if err != nil {
		fmt.Println("User creation error!\n\n", err)
	} else {
		fmt.Println("User created!")

		fmt.Println(userInfo.UserID)
		userInfo.UserID = "a$" + userInfo.UserID
		fmt.Println(userInfo.UserID)
	}

	//Create subuser
	fmt.Println("\nCreating subuser...")
	_, err = aa.SubUserCreate(context.Background(), &rgw.SubUserCreateModifyRequest{UID: userInfo.UserID, SubUser: "my-subuser", Access: "readwrite"})
	if err != nil {
		fmt.Println("Subuser creation error!\n\n", err)
	} else {
		fmt.Println("Subuser created!")
	}

	_, err = aa.SubUserCreate(context.Background(), &rgw.SubUserCreateModifyRequest{UID: userInfo.UserID, SubUser: "my-subuser2", Access: "readwrite"})
	gen := true
	time.Sleep(time.Second * 5)
	fmt.Println(aa.KeyCreate(context.Background(), &rgw.KeyCreateRequest{UID: "a$new-user", GenerateKey: &gen}))
	fmt.Println("WAITING")
	time.Sleep(time.Second * 5)
	aa.KeyCreate(context.Background(), &rgw.KeyCreateRequest{UID: "a$new-user", SubUser: "my-subuser", GenerateKey: &gen})
	fmt.Println("WAITING2")
	time.Sleep(time.Second * 5)

	//Set user quota
	fmt.Println("\nUpdating user quota...")
	quotaSettings := &rgw.QuotaSetRequest{
		UID:            userInfo.UserID,
		QuotaType:      "user",
		Enabled:        true,
		MaximumObjects: 100,
		MaximumSizeKb:  1 * 1000000, // 1GB
	}
	err = aa.QuotaSet(context.Background(), quotaSettings)
	if err != nil {
		fmt.Println("Updating quota error!\n\n", err)
	} else {
		fmt.Println("Quota updated!")
	}

	//Get user info
	fmt.Println("\nGetting user info...")
	userInfo, err = aa.UserInfo(context.Background(), userInfo.UserID, false)
	if err != nil {
		fmt.Println("Getting user info error!\n\n", err)
	} else {
		fmt.Println("User info retrieved!")
		fmt.Printf("\n%+v", userInfo)
		fmt.Printf("\n\nPause...\n\n")
	}

	return userInfo
}

func SwiftFunctionsTests(userInfo *rgw.UserInfoResponse) {
	// Create a connection
	fmt.Println("Swift Tests\n------------")
	c := swift.Connection{
		UserName: userInfo.SwiftKeys[0].User,
		ApiKey:   userInfo.SwiftKeys[0].SecretKey,
		AuthUrl:  swiftUrl,
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

func UserDeletionTests(userInfo *rgw.UserInfoResponse, secrets map[string]string) {
	//Initial connection
	fmt.Println("\n\nUser Deletion Tests\n------------")
	cfg := &rgw.Config{
		ClientConfig: rcl.ClientConfig{
			ClientTimeout: rcl.Duration(time.Second * 10),
		},
		ServerURL:       radosUrl,
		AdminPath:       radosAdminPath,
		AccessKeyID:     secrets["keyID"],
		SecretAccessKey: secrets["secretKey"],
	}

	aa, err := rgw.NewAdminAPI(cfg)
	if err != nil {
		fmt.Println("Admin API error\n\n", err)
	}

	//Delete user
	fmt.Println("Deleting user...")
	fmt.Println(userInfo.UserID)
	err = aa.UserRm(context.Background(), "a$"+userInfo.UserID, true)
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		fmt.Println("User deleted!")
	}
}
