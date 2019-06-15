package config

import (
	"reflect"
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
)

func newTestConfig() *Config {
	cfg := &Config{}
	serviceA := cfg.AddService("a", &dockerComposeConfig.Service{})
	serviceB := cfg.AddService("b", &dockerComposeConfig.Service{})
	serviceC := cfg.AddService("c", &dockerComposeConfig.Service{})
	serviceD := cfg.AddService("d", &dockerComposeConfig.Service{})
	serviceA.DockerComposeService.DependsOn = map[*dockerComposeConfig.Service]dockerComposeConfig.ServiceHealthiness{
		serviceB.DockerComposeService: dockerComposeConfig.ServiceHealthy,
	}
	serviceB.DockerComposeService.DependsOn = map[*dockerComposeConfig.Service]dockerComposeConfig.ServiceHealthiness{
		serviceC.DockerComposeService: dockerComposeConfig.ServiceHealthy,
		serviceD.DockerComposeService: dockerComposeConfig.ServiceHealthy,
	}
	return cfg
}

func TestAddToFilter(t *testing.T) {
	cfg := newTestConfig()

	// Since a depends on b, and b depends on c and d, we expect the result to contain all 4 apps.
	cfg.AddToFilter(cfg.FindServiceByName("a"))
	resultContainsAppA := cfg.MatchesFilter(cfg.FindServiceByName("a"))
	resultContainsAppB := cfg.MatchesFilter(cfg.FindServiceByName("b"))
	resultContainsAppC := cfg.MatchesFilter(cfg.FindServiceByName("c"))
	resultContainsAppD := cfg.MatchesFilter(cfg.FindServiceByName("d"))
	if !resultContainsAppA || !resultContainsAppB || !resultContainsAppC || !resultContainsAppD {
		t.Fail()
	}
}

func TestClearFilter(t *testing.T) {
	cfg := newTestConfig()
	cfg.AddToFilter(cfg.FindServiceByName("a"))
	cfg.ClearFilter()
	for _, service := range cfg.Services {
		if service.matchesFilter {
			t.Fail()
		}
	}
}

func TestAddService_ErrorDuplicateName(t *testing.T) {
	cfg := newTestConfig()
	defer func() {
		if err := recover(); err == nil {
			t.Fail()
		}
	}()
	cfg.AddService("a", &dockerComposeConfig.Service{})
}

func TestAddService_ErrorDockerComposeServiceInUse(t *testing.T) {
	cfg := newTestConfig()
	defer func() {
		if err := recover(); err == nil {
			t.Fail()
		}
	}()
	cfg.AddService("z", cfg.FindServiceByName("a").DockerComposeService)
}

func TestAddService_ErrorServiceHasDependsOn(t *testing.T) {
	cfg := newTestConfig()
	defer func() {
		if err := recover(); err == nil {
			t.Fail()
		}
	}()
	cfg.AddService("z", &dockerComposeConfig.Service{
		DependsOn: map[*dockerComposeConfig.Service]dockerComposeConfig.ServiceHealthiness{
			cfg.FindServiceByName("a").DockerComposeService: dockerComposeConfig.ServiceStarted,
		},
	})
}

var dockerComposeYmlInvalid = "/docker-compose.invalid.yml"
var dockerComposeYmlInvalidServiceName = "/docker-compose.invalid-service-name.yml"
var dockerComposeYmlInvalidXKubeCompose = "/docker-compose.invalid-x-kube-compose.yml"
var dockerComposeYmlValidPushImages = "/docker-compose.valid-push-images.yml"
var vfs fs.VirtualFileSystem = fs.NewInMemoryFileSystem(map[string]fs.InMemoryFile{
	dockerComposeYmlInvalid: {
		Content: []byte(`version: 'asdf'`),
	},
	dockerComposeYmlInvalidServiceName: {
		Content: []byte(`version: '2'
services:
  '!!':
    image: ubuntu:latest
`),
	},
	dockerComposeYmlInvalidXKubeCompose: {
		Content: []byte(`version: '2'
services:
  asdf:
    image: ubuntu:latest
    ports: [8080]
x-kube-compose:
  push_images: ""
`),
	},
	dockerComposeYmlValidPushImages: {
		Content: []byte(`version: '2'
x-kube-compose:
  push_images:
    docker_registry: 'my-docker-registry.example.com'
`),
	},
})

func withMockFS(cb func()) {
	orig := fs.OS
	defer func() {
		fs.OS = orig
	}()
	fs.OS = vfs
	cb()
}

func TestNew_Invalid(t *testing.T) {
	withMockFS(func() {
		_, err := New(util.NewString(dockerComposeYmlInvalid))
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func TestNew_InvalidServiceName(t *testing.T) {
	withMockFS(func() {
		_, err := New(util.NewString(dockerComposeYmlInvalidServiceName))
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func TestNew_InvalidXKubeCompose(t *testing.T) {
	withMockFS(func() {
		_, err := New(util.NewString(dockerComposeYmlInvalidXKubeCompose))
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func TestNew_ValidPushImages(t *testing.T) {
	withMockFS(func() {
		c, err := New(util.NewString(dockerComposeYmlValidPushImages))
		if err != nil {
			t.Error(err)
		} else {
			expected := &PushImagesConfig{
				DockerRegistry: "my-docker-registry.example.com",
			}
			if !reflect.DeepEqual(c.PushImages, expected) {
				t.Logf("pushImages1: %+v\n", c.PushImages)
				t.Logf("pushImages2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}
