package config

import (
	"context"
	"fmt"

	log "github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/docker/client"
	containerService "github.com/kube-compose/kube-compose/internal/pkg/container/service"
	buildahContainerService "github.com/kube-compose/kube-compose/internal/pkg/container/service/buildah"
	dockerContainerService "github.com/kube-compose/kube-compose/internal/pkg/container/service/docker"
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
	NameEscaped          string
	Ports                []Port
}

func (s *Service) Name() string {
	return s.DockerComposeService.Name
}

type ClusterImageStorage struct {
	Docker         *struct{}
	DockerRegistry *DockerRegistryClusterImageStorage
}

type Config struct {
	// All Kubernetes resources are named with "-"+EnvironmentID as a suffix,
	// and have an additional label "env="+EnvironmentID so that namespaces can be shared.
	EnvironmentID       string
	EnvironmentLabel    string
	KubeConfig          *rest.Config
	Namespace           string
	ClusterImageStorage ClusterImageStorage
	VolumeInitBaseImage *string

	Services map[string]*Service
}

type Port struct {
	Port int32
	// one of "udp", "tcp" and "sctp"
	Protocol string
}

func New(files []string) (*Config, error) {
	cfg := &Config{
		EnvironmentLabel: "env",
	}
	dcCfg, err := dockerComposeConfig.New(files)
	if err != nil {
		return nil, err
	}
	cfg.Services = map[string]*Service{}
	for name, dcService := range dcCfg.Services {
		if e := validation.IsDNS1123Subdomain(name); len(e) > 0 {
			return nil, fmt.Errorf("sorry, we do not support the potentially valid docker compose service named %s: %s", name, e[0])
		}
		service := &Service{
			DockerComposeService: dcService,
			NameEscaped:          util.EscapeName(name),
		}
		for _, portBinding := range dcService.Ports {
			service.Ports = append(service.Ports, Port{
				Protocol: portBinding.Protocol,
				Port:     portBinding.Internal,
			})
		}
		cfg.Services[name] = service
	}
	err = loadXKubeCompose(cfg, dcCfg.XProperties)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (cfg *Config) GetContainerService(ctx context.Context) (containerService.ContainerService, error) {
	dc, err := dockerClient.NewEnvClient()
	if err != nil {
		return nil, err
	}
	_, err = dc.Ping(ctx)
	if err == nil {
		log.Info("using docker daemon container service")
		return dockerContainerService.New(dc), nil
	}
	if !dockerClient.IsErrConnectionFailed(err) {
		return nil, err
	}
	sc, err := buildahContainerService.New()
	if err == nil {
		log.Info("using Buildah container service")
		return sc, nil
	}
	if !buildahContainerService.IsErrNotSupported(err) {
		return nil, err
	}
	return nil, fmt.Errorf("could not connect to docker daemon (is one running?), and cannot currently fall back to Buildah on non-Linux")
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
			log.Warn("a docker compose file has set \"x-kube-compose\".\"push_images\", but this functionality is deprecated. " +
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
func (cfg *Config) AddService(dockerComposeService *dockerComposeConfig.Service) *Service {
	service := cfg.Services[dockerComposeService.Name]
	if service != nil {
		panic("a docker compose service with that name is already registered")
	} else {
		if dockerComposeService.DependsOn != nil {
			panic("cannot add dockerComposeService that has dependencies")
		}
		service = &Service{
			DockerComposeService: dockerComposeService,
			NameEscaped:          util.EscapeName(dockerComposeService.Name),
		}
		if cfg.Services == nil {
			cfg.Services = map[string]*Service{}
		}
		cfg.Services[dockerComposeService.Name] = service
	}
	return service
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
				service2 := cfg.Services[d]
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
