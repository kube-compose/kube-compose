package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
)

type CanonicalComposeFile struct {
	Services map[string]*Service
	Version  string
}

type Service struct {
	DependsOn           map[*Service]ServiceHealthiness
	Entrypoint          []string
	Environment         map[string]string
	Healthcheck         *Healthcheck
	HealthcheckDisabled bool
	Image               string
	Ports               []Port
	ServiceName         string
	WorkingDir          string

	// helpers for ensureNoDependsOnCycle
	recStack bool
	visited  bool
}

type PushImagesConfig struct {
	DockerRegistry string `yaml:"docker_registry"`
}

type Config struct {
	CanonicalComposeFile CanonicalComposeFile
	EnvironmentID        string // All Kubernetes resources are named with "-"+EnvironmentID as a suffix, and have an additional label "env="+EnvironmentID so that namespaces can be shared.
	EnvironmentLabel     string
	KubeConfig           *rest.Config
	Namespace            string
	PushImages           *PushImagesConfig
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

	var versionHolder struct {
		Version string `yaml:"version"`
	}
	err = yaml.Unmarshal(data, &versionHolder)
	if err != nil {
		return nil, err
	}
	if versionHolder.Version != "2.1" {
		return nil, fmt.Errorf("unsupported docker-compose version")
	}

	var composeYAML composeYAML2_1
	err = yaml.Unmarshal(data, &composeYAML)
	if err != nil {
		return nil, err
	}

	var customYAML struct {
		Custom struct {
			PushImages *PushImagesConfig `yaml:"push_images"`
		} `yaml:"x-jompose"`
	}
	err = yaml.Unmarshal(data, &customYAML)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		CanonicalComposeFile: CanonicalComposeFile{
			Version: versionHolder.Version,
		},
		EnvironmentLabel: "env",
	}
	err = parseComposeYAML2_1(&composeYAML, &cfg.CanonicalComposeFile)
	if err != nil {
		return nil, err
	}

	for name := range cfg.CanonicalComposeFile.Services {
		if errors := validation.IsDNS1123Subdomain(name); len(errors) > 0 {
			return nil, fmt.Errorf("sorry, we do not support the potentially valid docker-compose service named %s: %s", name, errors[0])
		}
	}

	if customYAML.Custom.PushImages != nil {
		cfg.PushImages = customYAML.Custom.PushImages
	}

	return cfg, nil
}

// helper for defer in ensureNoDependsOnCycle
func (service *Service) clearRecStack() {
	service.recStack = false
}

// https://www.geeksforgeeks.org/detect-cycle-in-a-graph/
func ensureNoDependsOnCycle(service *Service) error {
	service.visited = true
	service.recStack = true
	defer service.clearRecStack()
	for dep := range service.DependsOn {
		if !dep.visited {
			err := ensureNoDependsOnCycle(dep)
			if err != nil {
				return err
			}
		} else if dep.recStack {
			return fmt.Errorf("service %s depends on %s, but this means there is a cyclic dependency, aborting", service.ServiceName, dep.ServiceName)
		}
	}
	return nil
}

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
func parseComposeYAML2_1(composeYAML *composeYAML2_1, dockerComposeFile *CanonicalComposeFile) error {
	n := len(composeYAML.Services)
	if n > 0 {
		dockerComposeFile.Services = make(map[string]*Service, n)
		for name, serviceYAML := range composeYAML.Services {
			service, err := parseServiceYAML2_1(&serviceYAML)
			if err != nil {
				return err
			}
			service.ServiceName = name
			dockerComposeFile.Services[name] = service
			for dependsOnService := range serviceYAML.DependsOn.Values {
				if _, ok := composeYAML.Services[dependsOnService]; !ok {
					return fmt.Errorf("service %s refers to a non-existing service in depends_on: %s", name, dependsOnService)
				}
			}
		}
		for name1, serviceYAML := range composeYAML.Services {
			service1 := dockerComposeFile.Services[name1]
			service1.DependsOn = map[*Service]ServiceHealthiness{}
			for name2, serviceHealthiness := range serviceYAML.DependsOn.Values {
				service2 := dockerComposeFile.Services[name2]
				service1.DependsOn[service2] = serviceHealthiness
			}
		}
		for _, service := range dockerComposeFile.Services {

			// Reset the visisted marker on each service. This is a precondition of ensureNoDependsOnCycle.
			for _, service := range dockerComposeFile.Services {
				service.visited = false
			}

			// Run the cycle detection algorithm...
			err := ensureNoDependsOnCycle(service)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseServiceYAML2_1(serviceYAML *serviceYAML2_1) (*Service, error) {
	service := &Service{
		Entrypoint: serviceYAML.Entrypoint.Values,
		Image:      serviceYAML.Image,
		WorkingDir: serviceYAML.WorkingDir,
	}

	ports, err := parsePorts(serviceYAML.Ports)
	if err != nil {
		return service, err
	}
	service.Ports = ports

	healthcheck, healthcheckDisabled, err := ParseHealthcheck(serviceYAML.Healthcheck)
	if err != nil {
		return service, err
	}
	service.Healthcheck = healthcheck
	service.HealthcheckDisabled = healthcheckDisabled

	service.Environment = make(map[string]string, len(serviceYAML.Environment.Values))
	for _, pair := range serviceYAML.Environment.Values {
		var value string
		if pair.Value == nil {
			var ok bool
			value, ok = os.LookupEnv(pair.Name)
			if !ok {
				continue
			}
		} else {
			value = *pair.Value
		}
		service.Environment[pair.Name] = value
	}

	return service, nil
}
