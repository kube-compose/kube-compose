package config

import (
	"gopkg.in/yaml.v2"
)

type Config struct {
	ComposeYamlData []byte
}

func New() (*Config, error) {
	cfg := &Config{}

	bla := struct{}{}
	err := yaml.Unmarshal([]byte{1,2}, &bla)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
