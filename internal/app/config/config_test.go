package config

import (
	"reflect"
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
)

func newTestConfig() *Config {
	cfg := &Config{}
	serviceA := cfg.AddService(&dockerComposeConfig.Service{
		Name: "a",
	})
	serviceB := cfg.AddService(&dockerComposeConfig.Service{
		Name: "b",
	})
	cfg.AddService(&dockerComposeConfig.Service{
		Name: "c",
	})
	cfg.AddService(&dockerComposeConfig.Service{
		Name: "d",
	})
	serviceA.DockerComposeService.DependsOn = map[string]dockerComposeConfig.ServiceHealthiness{
		"b": dockerComposeConfig.ServiceHealthy,
	}
	serviceB.DockerComposeService.DependsOn = map[string]dockerComposeConfig.ServiceHealthiness{
		"c": dockerComposeConfig.ServiceHealthy,
		"d": dockerComposeConfig.ServiceHealthy,
	}
	return cfg
}

func TestAddToFilter(t *testing.T) {
	cfg := newTestConfig()

	// Since a depends on b, and b depends on c and d, we expect the result to contain all 4 apps.
	cfg.AddToFilter(cfg.Services["a"])
	resultContainsAppA := cfg.MatchesFilter(cfg.Services["a"])
	resultContainsAppB := cfg.MatchesFilter(cfg.Services["b"])
	resultContainsAppC := cfg.MatchesFilter(cfg.Services["c"])
	resultContainsAppD := cfg.MatchesFilter(cfg.Services["d"])
	if !resultContainsAppA || !resultContainsAppB || !resultContainsAppC || !resultContainsAppD {
		t.Fail()
	}
}

func TestClearFilter(t *testing.T) {
	cfg := newTestConfig()
	cfg.AddToFilter(cfg.Services["a"])
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
	cfg.AddService(&dockerComposeConfig.Service{
		Name: "a",
	})
}

func TestAddService_ErrorDockerComposeServiceInUse(t *testing.T) {
	cfg := newTestConfig()
	defer func() {
		if err := recover(); err == nil {
			t.Fail()
		}
	}()
	cfg.AddService(cfg.Services["a"].DockerComposeService)
}

func TestAddService_ErrorServiceHasDependsOn(t *testing.T) {
	cfg := newTestConfig()
	defer func() {
		if err := recover(); err == nil {
			t.Fail()
		}
	}()
	cfg.AddService(&dockerComposeConfig.Service{
		DependsOn: map[string]dockerComposeConfig.ServiceHealthiness{
			"a": dockerComposeConfig.ServiceStarted,
		},
		Name: "z",
	})
}

var dockerComposeYmlInvalid = "/docker-compose.invalid.yml"
var dockerComposeYmlInvalidServiceName = "/docker-compose.invalid-service-name.yml"
var dockerComposeYmlInvalidXKubeCompose = "/docker-compose.invalid-x-kube-compose.yml"
var dockerComposeYmlValidPushImages = "/docker-compose.valid-push-images.yml"
var vfs fs.VirtualFileSystem = fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
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

func withMockFS2(vfsMock fs.VirtualFileSystem, cb func()) {
	orig := fs.OS
	defer func() {
		fs.OS = orig
	}()
	fs.OS = vfsMock
	cb()
}

func Test_New_Invalid(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{dockerComposeYmlInvalid})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_InvalidServiceName(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{dockerComposeYmlInvalidServiceName})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_InvalidXKubeCompose(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{dockerComposeYmlInvalidXKubeCompose})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_ValidPushImages(t *testing.T) {
	withMockFS(func() {
		c, err := New([]string{dockerComposeYmlValidPushImages})
		if err != nil {
			t.Error(err)
		} else {
			expected := ClusterImageStorage{
				DockerRegistry: &DockerRegistryClusterImageStorage{
					Host: "my-docker-registry.example.com",
				},
			}
			if !reflect.DeepEqual(c.ClusterImageStorage, expected) {
				t.Logf("pushImages1: %+v\n", c.ClusterImageStorage)
				t.Logf("pushImages2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}

func Test_New_MergeSuccess(t *testing.T) {
	file1 := "/xkubecomposemergesuccess1"
	file2 := "/xkubecomposemergesuccess2"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file1: {
			Content: []byte(`version: '2.4'
services:
  service1:
    image: ubuntu:latest
    environment:
      ENV: docker_desktop
x-kube-compose:
  cluster_image_storage:
    type: docker
`),
		},
		file2: {
			Content: []byte(`version: '2.4'
services:
  service1:
    environment:
      ENV: openshift
x-kube-compose:
  cluster_image_storage:
    type: docker_registry
    host: my-docker-registry.openshift-cluster.example.com
`),
		},
	}), func() {
		c, err := New([]string{
			file1,
			file2,
		})
		if err != nil {
			t.Error(err)
		} else {
			if c.Services["service1"].DockerComposeService.Environment["ENV"] != "openshift" {
				t.Fail()
			}
			expected := ClusterImageStorage{
				DockerRegistry: &DockerRegistryClusterImageStorage{
					Host: "my-docker-registry.openshift-cluster.example.com",
				},
			}
			if !reflect.DeepEqual(c.ClusterImageStorage, expected) {
				t.Logf("pushImages1: %+v\n", c.ClusterImageStorage)
				t.Logf("pushImages2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}

func Test_New_ClusterImageStorageDockerSuccess(t *testing.T) {
	file := "/dockersuccess"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file: {
			Content: []byte(`version: '2.4'
x-kube-compose:
  cluster_image_storage:
    type: docker
`),
		},
	}), func() {
		c, err := New([]string{file})
		if err != nil {
			t.Error(err)
		} else {
			expected := ClusterImageStorage{
				Docker: &struct{}{},
			}
			if !reflect.DeepEqual(c.ClusterImageStorage, expected) {
				t.Fail()
			}
		}
	})
}

func Test_New_ClusterImageStorageInvalidType(t *testing.T) {
	file := "/invalidtype"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file: {
			Content: []byte(`version: '2.4'
x-kube-compose:
  cluster_image_storage:
    type: invalid
`),
		},
	}), func() {
		_, err := New([]string{file})
		if err == nil {
			t.Fail()
		}
	})
}

func Test_New_ClusterImageStorageDockerRegistryMissingHost(t *testing.T) {
	file := "/dockerregistrymissinghost"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file: {
			Content: []byte(`version: '2.4'
x-kube-compose:
  cluster_image_storage:
    type: docker_registry
`),
		},
	}), func() {
		_, err := New([]string{file})
		if err == nil {
			t.Fail()
		}
	})
}

func Test_New_ClusterImageStorageDockerRegistrySuccess(t *testing.T) {
	file := "/dockerregistrysuccess"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file: {
			Content: []byte(`version: '2.4'
x-kube-compose:
  cluster_image_storage:
    type: docker_registry
    host: docker-registry-default.openshift-cluster.example.com
`),
		},
	}), func() {
		c, err := New([]string{file})
		if err != nil {
			t.Error(err)
		} else {
			expected := ClusterImageStorage{
				DockerRegistry: &DockerRegistryClusterImageStorage{
					Host: "docker-registry-default.openshift-cluster.example.com",
				},
			}
			if !reflect.DeepEqual(c.ClusterImageStorage, expected) {
				t.Fail()
			}
		}
	})
}

func Test_New_ClusterImageStoragePushImagesAlsoSpecified(t *testing.T) {
	file := "/pushimagesalsospecified"
	withMockFS2(fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
		file: {
			Content: []byte(`version: '2.4'
x-kube-compose:
  cluster_image_storage:
    type: docker
  push_images:
    docker_registry: docker-registry-default.openshift-cluster.example.com
`),
		},
	}), func() {
		_, err := New([]string{file})
		if err == nil {
			t.Fail()
		}
	})
}
