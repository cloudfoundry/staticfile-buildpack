package libbuildpack

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

type JSON struct {
}

func NewJSON() *JSON {
	return &JSON{}
}

func (j *JSON) Load(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	return nil
}

func (j *JSON) Write(dest string, obj interface{}) error {
	data, err := json.Marshal(&obj)
	if err != nil {
		return err
	}

	err = writeToFile(bytes.NewBuffer(data), dest, 0666)
	if err != nil {
		return err
	}
	return nil
}
