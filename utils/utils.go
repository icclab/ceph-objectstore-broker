package utils

import (
	"encoding/json"
	"io/ioutil"
)

func LoadJson(configPath string, i interface{}) error {
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
