package config

import (
	"fmt"

	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	dockerComposeConfig "github.com/jbrekelmans/kube-compose/pkg/docker/compose/config"
	"github.com/pkg/errors"
	"github.com/uber-go/mapdecode"
	"k8s.io/client-go/rest"
)

type PushImagesConfig struct {
	DockerRegistry string `mapdecode:"docker_registry"`
}

type Service struct {
	DockerComposeService *dockerComposeConfig.Service
	matchesFilter        bool
	Name                 string
	NameEscaped          string
	Ports                []Port
}

type Config struct {
	dockerComposeServices map[string]*dockerComposeConfig.Service

	// All Kubernetes resources are named with "-"+EnvironmentID as a suffix,
	// and have an additional label "env="+EnvironmentID so that namespaces can be shared.
	EnvironmentID    string
	EnvironmentLabel string

	KubeConfig *rest.Config
	Namespace  string
	PushImages *PushImagesConfig

	Services map[*dockerComposeConfig.Service]*Service
}

type Port struct {
	Port int32
	// one of "udp", "tcp" and "sctp"
	Protocol string
}

func (cfg *Config) FindServiceByName(name string) *Service {
	return cfg.FindService(cfg.dockerComposeServices[name])
}

func (cfg *Config) FindService(dockerComposeService *dockerComposeConfig.Service) *Service {
	return cfg.Services[dockerComposeService]
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func New(file *string) (*Config, error) {
	var files []string
	if file != nil {
		files = append(files, *file)
	}
	cfg := &Config{
		EnvironmentLabel: "env",
	}
	dcCfg, err := dockerComposeConfig.New(files)
	if err != nil {
		return nil, err
	}
	cfg.dockerComposeServices = dcCfg.Services
	cfg.Services = map[*dockerComposeConfig.Service]*Service{}
	for name, dcService := range dcCfg.Services {
		service := &Service{
			DockerComposeService: dcService,
			Name:                 name,
			NameEscaped:          util.EscapeName(name),
		}
		for _, portBinding := range dcService.Ports {
			service.Ports = append(service.Ports, Port{
				Protocol: portBinding.Protocol,
				Port:     portBinding.Internal,
			})
		}
		cfg.Services[dcService] = service
	}
	var custom struct {
		Custom struct {
			PushImages *PushImagesConfig `mapdecode:"push_images"`
		} `mapdecode:"x-kube-compose"`
	}
	err = mapdecode.Decode(&custom, dcCfg.XProperties, mapdecode.IgnoreUnused(true))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error while parsing x-kube-compose of %#v", *file))
	}

	if custom.Custom.PushImages != nil {
		cfg.PushImages = custom.Custom.PushImages
	}

	return cfg, nil
}

// AddService adds a service to this configuration.
func (cfg *Config) AddService(name string, dockerComposeService *dockerComposeConfig.Service) *Service {
	service1 := cfg.FindServiceByName(name)
	service2 := cfg.FindService(dockerComposeService)
	if service1 != nil || service2 != nil {
		if service1 == nil {
			panic("dockerComposeService was previously registered with a different name")
		} else if service2 == nil {
			panic("a service with name already exists")
		}
	} else {
		if len(dockerComposeService.DependsOn) > 0 {
			panic("cannot add dockerComposeService that has dependencies")
		}
		service1 = &Service{
			DockerComposeService: dockerComposeService,
			Name:                 name,
			NameEscaped:          util.EscapeName(name),
		}
		if cfg.Services == nil {
			cfg.Services = map[*dockerComposeConfig.Service]*Service{}
		}
		cfg.Services[dockerComposeService] = service1
		if cfg.dockerComposeServices == nil {
			cfg.dockerComposeServices = map[string]*dockerComposeConfig.Service{}
		}
		cfg.dockerComposeServices[name] = dockerComposeService
	}
	return service1
}

// MatchesFilter determines whether a service (by name) matches the current filter.
func (cfg *Config) MatchesFilter(service *Service) bool {
	return service.matchesFilter
}

// ClearFilter sets the current filter to match no service.
func (cfg *Config) ClearFilter() {
	for _, service := range cfg.Services {
		service.matchesFilter = false
	}
}

// AddToFilter adds service and its (in)direct dependencies (based on depends_on) to the set of services matched by
// the current filter.
func (cfg *Config) AddToFilter(service *Service) {
	queue := []*Service{
		service,
	}
	n := 1
	for n > 0 {
		n--
		service1 := queue[n]
		if !service1.matchesFilter {
			service1.matchesFilter = true
			for d := range service1.DockerComposeService.DependsOn {
				service2 := cfg.FindService(d)
				if n < len(queue) {
					queue[n] = service2
				} else {
					queue = append(queue, service2)
				}
				n++
			}
		}
	}
}
