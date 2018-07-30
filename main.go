package main

// https://ned.cloudlab.zhaw.ch:6780/swift/v1
import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/ncw/swift"
	"github.com/pivotal-cf/brokerapi"
	"github.engineering.zhaw.ch/kaio/swift-go-broker/broker"
	"net/http"
	"os"
)

func main() {
	s := &broker.Broker{ServiceID: "1234-4321-abcd-fghi", PlanID: "4321-1234-abcd-fghi"}
	creds := brokerapi.BrokerCredentials{Username: "admin", Password: "admin"}

	logger := lager.NewLogger("S3-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	logger.Debug("Starting")
	handler := brokerapi.New(s, logger, creds)

	http.Handle("/", handler)
	logger.Debug("Handling requests")

	test()

	logger.Debug("Listen and serve on port: 8080")
	_ = http.ListenAndServe(":8080", nil)
}

func test() {
	// Create a connection
	c := swift.Connection{
		UserName: "kaio",
		ApiKey:   "account-password",
		AuthUrl:  "https://ned.cloudlab.zhaw.ch:5000/v3",
		Domain:   "Default",
	}

	// Authenticate
	err := c.Authenticate()
	if err != nil {
		panic(err)
	}

	// List all the containers
	containers, err := c.ContainerNames(nil)
	fmt.Println(containers)

	contName := "kaio-test-container"
	if _, _, e := c.Container(contName); e != nil {

		_ = c.ContainerCreate(contName, nil)
		fmt.Println("Container:'", contName, "' created")
	}

	// f, e := c.ObjectCreate(contName, "test-file.txt", false, "", "", nil)
}
