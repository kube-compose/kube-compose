package config

import (
	"gopkg.in/yaml.v2"
)

type Config struct {
	ComposeYamlData byte[]
}

func New() (*config, error) {
	cfg = &Config{}

	// yaml.Unmarshal()

	return cfg, nil
}