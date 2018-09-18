package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

//hi
func main() {
	varsFile, err := os.Open("vars-file.yml")
	if err != nil {
		fmt.Println("Failed to open file 'vars-file.yml'.", err)
		return
	}
	defer varsFile.Close()

	configFile, err := os.Create("deployment-configs/k8s/config-map.yml")
	if err != nil {
		fmt.Println("Failed to open/create file 'deployment-configs/k8s/config-map.yml'.", err)
		return
	}
	defer configFile.Close()
	configWriter := bufio.NewWriter(configFile)
	fmt.Fprintln(configWriter, "apiVersion: v1")
	fmt.Fprintln(configWriter, "kind: ConfigMap")
	fmt.Fprintln(configWriter, "metadata:")
	fmt.Fprintln(configWriter, "  name: cosb-env")
	fmt.Fprintln(configWriter, "data:")

	secretFile, err := os.Create("deployment-configs/k8s/secret.yml")
	if err != nil {
		fmt.Println("Failed to open/create file 'deployment-configs/k8s/secret.yml'.", err)
		return
	}
	defer secretFile.Close()
	secretWriter := bufio.NewWriter(secretFile)
	fmt.Fprintln(secretWriter, "apiVersion: v1")
	fmt.Fprintln(secretWriter, "kind: Secret")
	fmt.Fprintln(secretWriter, "metadata:")
	fmt.Fprintln(secretWriter, "  name: cosb-secret")
	fmt.Fprintln(secretWriter, "type: Opaque")
	fmt.Fprintln(secretWriter, "data:")

	envFile, err := os.Create("tests/tests.env")
	if err != nil {
		fmt.Println("Failed to open/create file 'tests/tests.env'.", err)
		return
	}
	defer envFile.Close()
	envWriter := bufio.NewWriter(envFile)

	//Read vars and write them to the file
	scanner := bufio.NewScanner(varsFile)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || string(strings.TrimSpace(line)[0]) == "#" || !strings.Contains(line, ":") {
			continue
		}

		entry := strings.SplitN(line, ":", 2)
		fmt.Fprintln(configWriter, "  "+strings.ToUpper(entry[0])+":", strings.TrimSpace(entry[1]))
		fmt.Fprintln(envWriter, "export "+strings.ToUpper(entry[0])+"="+strings.TrimSpace(entry[1]))

		//Write the broker creds
		if entry[0] == "broker_username" || entry[0] == "broker_password" {
			key := strings.Replace(entry[0], "broker_", "", 1)
			val := strings.TrimSuffix(strings.TrimSpace(entry[1]), "\"")[1:]
			fmt.Fprintln(secretWriter, "  "+key+":", base64.StdEncoding.EncodeToString([]byte(val)))
		}
	}

	configWriter.Flush()
	secretWriter.Flush()
	envWriter.Flush()
}
