package utils

import (
	"encoding/json"
	"io/ioutil"
)

//Decodes a json string into the passed struct
//'i' must be passed by reference
func LoadJson(jsonString string, i interface{}) error {
	return json.Unmarshal([]byte(jsonString), i)
}

//Decodes a json file into the passed struct
//'i' must be passed by reference
func LoadJsonFromFile(configPath string, i interface{}) error {
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, i)
	if err != nil {
		return err
	}

	return nil
}
