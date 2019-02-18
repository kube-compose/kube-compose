package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type DockerComposeFile struct {
	Services map[string]Service
	Version string
}

type Service struct{
	Environment map[string]string
	Healthcheck *Healthcheck
	HealthcheckDisabled bool
	Image string
	Ports []Port
	WorkingDir string
}

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
type Config struct {
	DockerComposeFile DockerComposeFile
}

func New() (*Config, error) {
	data, err := ioutil.ReadFile("docker-compose.yml")
	if err != nil {
		if os.IsNotExist(err) {
			data, err = ioutil.ReadFile("docker-compose.yaml")
		}
		if err != nil {
			return nil, err
		}
	}

	var versionHolder struct{
		Version string `yaml:"version"`
	}
	err = yaml.Unmarshal(data, &versionHolder)
	if err != nil {
		return nil, err
	}
	if versionHolder.Version != "2.1" {
		return nil, fmt.Errorf("Unsupported docker-compose version")	
	}

	var composeYAML composeYAML2_1
	err = yaml.Unmarshal(data, &composeYAML)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DockerComposeFile: DockerComposeFile{
			Version: versionHolder.Version,
		},
	}
	err = parseComposeYAML2_1(&composeYAML, &cfg.DockerComposeFile)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseComposeYAML2_1(composeYAML *composeYAML2_1, dockerComposeFile *DockerComposeFile) error {
	n := len(composeYAML.Services)
	if n > 0 {
		dockerComposeFile.Services = make(map[string]Service, n)
		for name, serviceYAML := range composeYAML.Services {
			service, err := parseServiceYAML2_1(&serviceYAML)
			if err != nil {
				return err
			}
			dockerComposeFile.Services[name] = service
		}
	}
	return nil
}

func parseServiceYAML2_1 (serviceYAML *serviceYAML2_1) (Service, error) {
	service := Service{
		Environment: serviceYAML.Environment,
		Image: serviceYAML.Image,
		WorkingDir: serviceYAML.WorkingDir,
	}

	ports, err := parsePorts(serviceYAML.Ports)
	if err != nil {
		return service, err
	}
	service.Ports = ports

	healthcheck, healthcheckDisabled, err := parseHealthcheck(serviceYAML.Healthcheck)
	if err != nil {
		return service, err
	}
	service.Healthcheck = healthcheck
	service.HealthcheckDisabled = healthcheckDisabled

	return service, nil
}
