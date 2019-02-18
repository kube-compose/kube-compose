package config

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"gopkg.in/yaml.v2"
)

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
type Config struct {
	ComposeYaml struct{
		Version  string `yaml:"version"`
		Services map[string]struct{
			Build struct{
				Context string `yaml:"context"`
				Dockerfile string `yaml:"dockerfile"`
			} `yaml:"build"`
			Environment map[string]string `yaml:"environment"`
			Image string `yaml:"image"`
			Ports []string `yaml:"ports"`
			Volumes []string `yaml:"volumes"`
			WorkingDir string `yaml:"working_dir"`
		} `yaml:"services"`
		Volumes map[string]interface{} `yaml:"volumes"`
	}
}

func New() (*Config, error) {
	cfg := &Config{}
	data, err := ioutil.ReadFile("docker-compose.yml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cfg.ComposeYaml)
	if err != nil {
		return nil, err
	}
	if cfg.ComposeYaml.Version != "2.1" {
		return nil, fmt.Errorf("Unsupported docker-compose version")	
	}
	return cfg, nil
}

// https://docs.docker.com/compose/compose-file/compose-file-v2/
// ports:
//  - "3000"
//  - "3000-3005"
//  - "8000:8000"
//  - "9090-9091:8080-8081"
//  - "49100:22"
//  - "127.0.0.1:8001:8001"
//  - "127.0.0.1:5000-5010:5000-5010"
//  - "6060:6060/udp"
//  - "12400-12500:1240"



type Port struct {
	ContainerPort int32
	ExternalPort int32
	Protocol string
}

func ParsePorts (inPorts []string) ([]Port, error) {
	outPorts := []Port{}
	for _, portStr := range inPorts {
		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			return outPorts, err
		}
		if port >= 65536 {
			return outPorts, fmt.Errorf("port must be < 65536 but got %d", port)
		}
		outPorts = append(outPorts, Port{
			ContainerPort: int32(port),
			ExternalPort: int32(port),
			Protocol: "TCP",
		})
	}
	return outPorts, nil
} 
