package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/uber-go/mapdecode"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
)

var (
	v1   = version.Must(version.NewVersion("1"))
	v2_1 = version.Must(version.NewVersion("2.1"))
	v3_1 = version.Must(version.NewVersion("3.1"))
	v3_3 = version.Must(version.NewVersion("3.3"))
)

// TODO https://github.com/jbrekelmans/kube-compose/issues/11 remove this type
type genericMap map[interface{}]interface{}

type CanonicalComposeFile struct {
	Services map[string]*Service
	Version  *version.Version
}

type Service struct {
	DependsOn           map[*Service]ServiceHealthiness
	Entrypoint          []string
	Environment         map[string]string
	Healthcheck         *Healthcheck
	HealthcheckDisabled bool
	Image               string
	Ports               []PortBinding
	ServiceName         string
	WorkingDir          string

	// helpers for ensureNoDependsOnCycle
	recStack bool
	visited  bool
}

type PushImagesConfig struct {
	DockerRegistry string `mapdecode:"docker_registry"`
}

type Config struct {
	CanonicalComposeFile CanonicalComposeFile
	// All Kubernetes resources are named with "-"+EnvironmentID as a suffix,
	// and have an additional label "env="+EnvironmentID so that namespaces can be shared.
	EnvironmentID    string
	EnvironmentLabel string
	KubeConfig       *rest.Config
	Namespace        string
	PushImages       *PushImagesConfig
	// A filter of the docker compose services to start. Transitive dependencies of filtered are always started, even if they themselves
	// are not filtered. If the map is empty all services will be started.
	Services map[string]bool
	Detach   bool
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func New() (*Config, error) {
	fileName := "docker-compose.yml"
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			fileName = "docker-compose.yaml"
			data, err = ioutil.ReadFile(fileName)
		}
		if err != nil {
			return nil, err
		}
	}

	var dataMap genericMap
	err = yaml.Unmarshal(data, &dataMap)
	if err != nil {
		return nil, err
	}

	var ver *version.Version
	verRaw, hasVer := dataMap["version"]
	if !hasVer {
		ver = v1
	} else if verStr, ok := verRaw.(string); ok {
		ver, err = version.NewVersion(verStr)
		if err != nil {
			return nil, fmt.Errorf("file %#v has an invalid version: %#v", fileName, verStr)
		}
	} else {
		return nil, fmt.Errorf("file %#v has a version that is not a string", fileName)
	}

	// Substitute variables with environment variables.
	err = InterpolateConfig(fileName, dataMap, func(name string) (string, bool) {
		val, found := os.LookupEnv(name)
		return val, found
	}, ver)
	if err != nil {
		return nil, err
	}

	var composeFile composeFile2_1
	err = mapdecode.Decode(&composeFile, dataMap, mapdecode.IgnoreUnused(true))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error while parsing docker compose %#v", fileName))
	}

	var custom struct {
		Custom struct {
			PushImages *PushImagesConfig `mapdecode:"push_images"`
		} `mapdecode:"x-kube-compose"`
	}
	err = mapdecode.Decode(&custom, dataMap, mapdecode.IgnoreUnused(true))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error while parsing x-kube-compose of %#v", fileName))
	}

	cfg := &Config{
		CanonicalComposeFile: CanonicalComposeFile{
			Version: ver,
		},
		EnvironmentLabel: "env",
	}
	err = parseCompose2_1(&composeFile, &cfg.CanonicalComposeFile)
	if err != nil {
		return nil, err
	}

	for name := range cfg.CanonicalComposeFile.Services {
		if e := validation.IsDNS1123Subdomain(name); len(e) > 0 {
			return nil, fmt.Errorf("sorry, we do not support the potentially valid docker-compose service named %s: %s", name, e[0])
		}
	}

	if custom.Custom.PushImages != nil {
		cfg.PushImages = custom.Custom.PushImages
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
			return fmt.Errorf("service %s depends on %s, but this means there is a cyclic dependency, aborting",
				service.ServiceName, dep.ServiceName)
		}
	}
	return nil
}

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func parseCompose2_1(composeYAML *composeFile2_1, dockerComposeFile *CanonicalComposeFile) error {
	n := len(composeYAML.Services)
	if n > 0 {
		dockerComposeFile.Services = make(map[string]*Service, n)
		for name, serviceYAML := range composeYAML.Services {
			service, err := parseServiceYAML2_1(serviceYAML)
			if err != nil {
				return err
			}
			service.ServiceName = name
			dockerComposeFile.Services[name] = service
			if serviceYAML.DependsOn != nil {
				for dependsOnService := range serviceYAML.DependsOn.Values {
					if _, ok := composeYAML.Services[dependsOnService]; !ok {
						return fmt.Errorf("service %s refers to a non-existing service in depends_on: %s", name, dependsOnService)
					}
				}
			}
		}
		for name1, serviceYAML := range composeYAML.Services {
			service1 := dockerComposeFile.Services[name1]
			service1.DependsOn = map[*Service]ServiceHealthiness{}
			if serviceYAML.DependsOn != nil {
				for name2, serviceHealthiness := range serviceYAML.DependsOn.Values {
					service2 := dockerComposeFile.Services[name2]
					service1.DependsOn[service2] = serviceHealthiness
				}
			}
		}
		for _, service := range dockerComposeFile.Services {

			// Reset the visited marker on each service. This is a precondition of ensureNoDependsOnCycle.
			for _, service := range dockerComposeFile.Services {
				service.visited = false
			}

			// Run the cycle detection algorithm...
			err := ensureNoDependsOnCycle(service)
			if err != nil {
				return err
			}
		}

		// Handle extends, cannot extend a service that has depends_on
		for name, serviceYAML := range composeYAML.Services {
			if serviceYAML.Extends == nil {
				continue
			}
			if serviceYAML.Extends.File != nil {
				// TODO https://github.com/jbrekelmans/kube-compose/issues/43
				return fmt.Errorf("extends with file is not supported")
			}
			extendedServiceYAML, ok := composeYAML.Services[serviceYAML.Extends.Service]
			if !ok {
				return fmt.Errorf("service %s refers to a non-existing service in extends: %s", name, serviceYAML.Extends.Service)
			}
			if extendedServiceYAML.DependsOn != nil {
				return fmt.Errorf("cannot extend service %s: services with 'depends_on' cannot be extended", serviceYAML.Extends.Service)
			}
			// TODO check for links, volumes_from, net and network_mode as per:
			// https://github.com/docker/compose/blob/master/compose/config/config.py#L695

			service := dockerComposeFile.Services[name]
			extendedService := dockerComposeFile.Services[serviceYAML.Extends.Service]
			// Perform merge
			merge(service, extendedService)
			// TODO https://github.com/docker/compose/blob/master/compose/config/config.py#L1092
		}
	}
	return nil
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func parseServiceYAML2_1(serviceYAML *service2_1) (*Service, error) {
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
		if pair.Name == "" {
			return nil, fmt.Errorf("invalid environment variable: %s", pair.Name)
		}
		switch {
		case pair.Value == nil:
			var ok bool
			if value, ok = os.LookupEnv(pair.Name); !ok {
				continue
			}
		case pair.Value.StringValue != nil:
			value = *pair.Value.StringValue
		case pair.Value.IntValue != nil:
			value = strconv.Itoa(*pair.Value.IntValue)
		case pair.Value.FloatValue != nil:
			value = strconv.FormatFloat(*pair.Value.FloatValue, 'g', -1, 64)
		default:
			// Environment variables with null values in the YAML are ignored.
			// This was tested with docker-compose.null-env.yml.
			continue
		}
		service.Environment[pair.Name] = value
	}
	return service, nil
}
