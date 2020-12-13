package main

import (
	"io/ioutil"

	yml "gopkg.in/yaml.v3"
)

type config struct {
	Targets []target
}

type target struct {
	Addr     string
	Interval string
	Labels   map[string]string
}

func getConfig(filename string) (*config, error) {
	if len(filename) < 1 {
		return &config{Targets: []target{}}, nil
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &config{}
	err = yml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
