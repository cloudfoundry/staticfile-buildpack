package libbuildpack

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

type JSON interface {
	Load(file string, obj interface{}) error
	Write(dest string, obj interface{}) error
}

type jsonStruct struct {
}

func NewJSON() JSON {
	return &jsonStruct{}
}

func (y *jsonStruct) Load(file string, obj interface{}) error {
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

func (y *jsonStruct) Write(dest string, obj interface{}) error {
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
