package config

import (
	"fmt"

	"github.com/kube-compose/kube-compose/internal/pkg/util"
	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
	"github.com/pkg/errors"
	"github.com/uber-go/mapdecode"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
)

type DockerRegistryClusterImageStorage struct {
	Host string
}

type Service struct {
	DockerComposeService *dockerComposeConfig.Service
	matchesFilter        bool
	Name                 string
	NameEscaped          string
	Ports                []Port
}

type ClusterImageStorage struct {
	Docker         *struct{}
	DockerRegistry *DockerRegistryClusterImageStorage
}

type Config struct {
	dockerComposeServices map[string]*dockerComposeConfig.Service

	// All Kubernetes resources are named with "-"+EnvironmentID as a suffix,
	// and have an additional label "env="+EnvironmentID so that namespaces can be shared.
	EnvironmentID       string
	EnvironmentLabel    string
	KubeConfig          *rest.Config
	Namespace           string
	ClusterImageStorage ClusterImageStorage
	VolumeInitBaseImage *string

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

func New(files []string) (*Config, error) {
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
		if e := validation.IsDNS1123Subdomain(name); len(e) > 0 {
			return nil, fmt.Errorf("sorry, we do not support the potentially valid docker compose service named %s: %s", name, e[0])
		}
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
	err = loadXKubeCompose(cfg, dcCfg.XProperties)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

type clusterImageStorage struct {
	Type string  `mapdecode:"type"`
	Host *string `mapdecode:"host"`
}

type xKubeCompose struct {
	XKubeCompose struct {
		ClusterImageStorage *clusterImageStorage `mapdecode:"cluster_image_storage"`
		PushImages          *struct {
			DockerRegistry string `mapdecode:"docker_registry"`
		} `mapdecode:"push_images"`
		VolumeInitBaseImage *string `mapdecode:"volume_init_base_image"`
	} `mapdecode:"x-kube-compose"`
}

func loadXKubeCompose(cfg *Config, xPropertiesSlice []dockerComposeConfig.XProperties) error {
	for i := len(xPropertiesSlice) - 1; i >= 0; i-- {
		var x xKubeCompose
		err := mapdecode.Decode(&x, xPropertiesSlice[i], mapdecode.IgnoreUnused(true))
		if err != nil {
			return errors.Wrap(err, "error while parsing \"x-kube-compose\" of a docker compose file")
		}
		if x.XKubeCompose.ClusterImageStorage != nil {
			if x.XKubeCompose.PushImages != nil {
				return fmt.Errorf("a docker compose file cannot set both \"x-kube-compose\".\"push_images\" and \"x-kube-compose\"." +
					"\"cluster_image_storage\"")
			}
			err = loadClusterImageStorage(cfg, x.XKubeCompose.ClusterImageStorage)
			if err != nil {
				return err
			}
		} else if x.XKubeCompose.PushImages != nil {
			fmt.Println("WARNING: a docker compose file has set \"x-kube-compose\".\"push_images\", but this functionality is deprecated. " +
				"See https://github.com/kube-compose/kube-compose.")
			cfg.ClusterImageStorage.Docker = nil
			cfg.ClusterImageStorage.DockerRegistry = &DockerRegistryClusterImageStorage{
				Host: x.XKubeCompose.PushImages.DockerRegistry,
			}
		}
		cfg.VolumeInitBaseImage = x.XKubeCompose.VolumeInitBaseImage
	}
	return nil
}

func loadClusterImageStorage(cfg *Config, v *clusterImageStorage) error {
	cfg.ClusterImageStorage.Docker = nil
	cfg.ClusterImageStorage.DockerRegistry = nil
	switch v.Type {
	case "docker":
		cfg.ClusterImageStorage.Docker = &struct{}{}
	case "docker_registry":
		if v.Host == nil {
			return fmt.Errorf("a docker compose file is missing a required value at \"x-kube-compose\".\"cluster_image_storage\"." +
				"\"host\"")
		}
		cfg.ClusterImageStorage.DockerRegistry = &DockerRegistryClusterImageStorage{
			Host: *v.Host,
		}
	default:
		return fmt.Errorf("a docker compose file has an invalid value at \"x-kube-compose\".\"cluster_image_storage\".\"type\": " +
			"value must be one of \"docker\" and \"docker_registry\"")
	}
	return nil
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
		if dockerComposeService.DependsOn != nil {
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
