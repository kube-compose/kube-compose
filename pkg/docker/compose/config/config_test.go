package config

import (
	"fmt"
	"reflect"
	"testing"

	fsPackage "github.com/jbrekelmans/kube-compose/internal/pkg/fs"
	"github.com/pkg/errors"
)

const testDockerComposeYml = "docker-compose.yaml"
const testDockerComposeYmlIOError = "docker-compose.io-error.yaml"
const testDockerComposeYmlInvalidVersion = "docker-compose.invalid-version.yml"
const testDockerComposeYmlInterpolationIssue = "docker-compose.interpolation-issue.yml"
const testDockerComposeYmlDecodeIssue = "docker-compose.decode-issue.yml"
const testDockerComposeYmlExtends = "docker-compose.extends.yml"
const testDockerComposeYmlExtendsCycle = "docker-compose.extends-cycle.yml"

var mockFileSystem = fsPackage.MockFileSystem(map[string]fsPackage.MockFile{
	testDockerComposeYml: {
		Content: []byte(`testservice:
  image: ubuntu:latest
`),
	},
	testDockerComposeYmlIOError: {
		Error: errors.New("unknown error 1"),
	},
	testDockerComposeYmlInvalidVersion: {
		Content: []byte("version: ''"),
	},
	testDockerComposeYmlInterpolationIssue: {
		Content: []byte(`version: '2'
services:
  testservice:
    image: '$'
`),
	},
	testDockerComposeYmlDecodeIssue: {
		Content: []byte(`version: '2'
services:
  testservice:
    environment: 3
`),
	},
	testDockerComposeYmlExtends: {
		Content: []byte(`version: '2'
services:
  service1:
    environment:
      KEY1: VALUE1
    extends:
      service: service2
  service2:
    environment:
      KEY2: VALUE2
    extends:
      file: '` + testDockerComposeYml + `'
			service: testservice
  service3:
    extends:
      file: '` + testDockerComposeYml + `'
      service: testservice
`),
	},
	testDockerComposeYmlExtendsCycle: {
		Content: []byte(`version: '2'
services:
  service1:
    extends:
      service: service2
  service2:
    extends:
      service: service3
  service3:
    extends:
      service: service2
`),
	},
})

var mockFileSystemStandardFileError = fsPackage.MockFileSystem(map[string]fsPackage.MockFile{
	"docker-compose.yml": {
		Error: errors.New("unknown error 2"),
	},
})

func withMockFS(cb func()) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystem
	cb()
}

func newTestConfigLoader(env map[string]string) *configLoader {
	c := &configLoader{
		environmentGetter:     mapValueGetter(env),
		loadResolvedFileCache: map[string]*loadResolvedFileCacheItem{},
	}
	return c
}

func TestConfigLoaderParseEnvironment_Success(t *testing.T) {
	name1 := "CFGLOADER_PARSEENV_VAR1"
	value1 := "CFGLOADER_PARSEENV_VAL1"
	name2 := "CFGLOADER_PARSEENV_VAR2"
	name3 := "CFGLOADER_PARSEENV_VAR3"
	name4 := "CFGLOADER_PARSEENV_VAR4"
	input := []environmentNameValuePair{
		{
			Name: name1,
		},
		{
			Name: name2,
			Value: &environmentValue{
				StringValue: new(string),
			},
		},
		{
			Name: name3,
			Value: &environmentValue{
				Int64Value: new(int64),
			},
		},
		{
			Name: name4,
			Value: &environmentValue{
				FloatValue: new(float64),
			},
		},
		{
			Name:  "CFGLOADER_PARSEENV_VAR5",
			Value: &environmentValue{},
		},
		{
			Name: "CFGLOADER_PARSEENV_VAR6",
		},
	}
	c := newTestConfigLoader(map[string]string{
		name1: value1,
	})
	output, err := c.parseEnvironment(input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(output, map[string]string{
		name1: value1,
		name2: "",
		name3: "0",
		name4: "0",
	}) {
		t.Error(output)
	}
}
func TestConfigLoaderParseEnvironment_InvalidName(t *testing.T) {
	input := []environmentNameValuePair{
		{
			Name: "",
		},
	}
	c := newTestConfigLoader(nil)
	_, err := c.parseEnvironment(input)
	if err == nil {
		t.Fail()
	}
}

func TestConfigLoaderLoadFile_Success(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		cfParsed, err := c.loadFile(testDockerComposeYml)
		if err != nil {
			t.Error(err)
		} else {
			if !cfParsed.version.Equal(v1) {
				t.Fail()
			}
			if len(cfParsed.xProperties) != 0 {
				t.Fail()
			}
			expected := map[string]*composeFileParsedService{
				"testservice": {
					dependsOn: map[string]ServiceHealthiness{},
					service: &Service{
						DependsOn:   map[*Service]ServiceHealthiness{},
						Entrypoint:  []string{},
						Environment: map[string]string{},
						Image:       "ubuntu:latest",
						Ports:       []PortBinding{},
					},
				},
			}
			assertComposeFileServicesEqual(t, cfParsed.services, expected)
		}
	})
}

func assertComposeFileServicesEqual(t *testing.T, services1, services2 map[string]*composeFileParsedService) {
	if len(services1) != len(services2) {
		t.Fail()
	}
	for name, service1 := range services1 {
		service2 := services2[name]
		if service2 == nil {
			t.Fail()
		} else {
			if len(service1.dependsOn) > 0 || len(service2.dependsOn) > 0 {
				t.Fail()
			}
			if service1.extends != nil || service2.extends != nil {
				t.Fail()
			}
			assertServicesEqual(t, service1.service, service2.service)
		}
	}
}

func assertServiceMapsEqual(t *testing.T, services1, services2 map[string]*Service) {
	if len(services1) != len(services2) {
		t.Fail()
	}
	for name, service1 := range services1 {
		service2 := services2[name]
		if service2 == nil {
			t.Fail()
		} else {
			assertServicesEqual(t, service1, service2)
		}
	}
}

func assertServicesEqual(t *testing.T, service1, service2 *Service) {
	if service1.Restart != service2.Restart {
		t.Fail()
	}
	if service1.WorkingDir != service2.WorkingDir {
		t.Fail()
	}
	if (service1.User != service2.User) || (service1.User != nil && *service1.User != *service2.User) {
		t.Fail()
	}
	if service1.HealthcheckDisabled != service2.HealthcheckDisabled {
		t.Fail()
	}
	if service1.Healthcheck != nil || service2.Healthcheck != nil {
		t.Fail()
	}
	assertServicesEqualContinued(t, service1, service2)
}

func portsIsSubsetOf(ports1, ports2 []PortBinding) bool {
	for _, port1 := range ports1 {
		found := false
		for _, port2 := range ports2 {
			if port1 == port2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func arePortsEqual(ports1, ports2 []PortBinding) bool {
	if len(ports1) != len(ports2) {
		return false
	}
	return portsIsSubsetOf(ports1, ports2) && portsIsSubsetOf(ports2, ports1)
}

func assertServicesEqualContinued(t *testing.T, service1, service2 *Service) {
	if !arePortsEqual(service1.Ports, service2.Ports) {
		t.Logf("ports1: %+v\n", service1.Ports)
		t.Logf("ports2: %+v\n", service2.Ports)
		t.Fail()
	}
	if !reflect.DeepEqual(service1.Environment, service2.Environment) {
		t.Logf("env1: %+v\n", service1.Environment)
		t.Logf("env2: %+v\n", service2.Environment)
		t.Fail()
	}
	if !areStringSlicesEqual(service1.Entrypoint, service2.Entrypoint) {
		t.Logf("entrypoint1: %+v\n", service1.Entrypoint)
		t.Logf("entrypoint2: %+v\n", service2.Entrypoint)
		t.Fail()
	}
	if len(service1.DependsOn) > 0 || len(service2.DependsOn) > 0 {
		t.Logf("dependsOn1: %+v\n", service1.DependsOn)
		t.Logf("dependsOn2: %+v\n", service2.DependsOn)
		t.Fail()
	}
}

func areStringSlicesEqual(slice1, slice2 []string) bool {
	n := len(slice1)
	if n != len(slice2) {
		return false
	}
	for i := 0; i < n; i++ {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}

func TestConfigLoaderLoadFile_Error(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadFile(testDockerComposeYmlIOError)
		if err == nil {
			t.Fail()
		}
	})
}

func TestConfigLoaderLoadResolvedFile_Caching(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		cfParsed1, err := c.loadResolvedFile(testDockerComposeYml)
		if err != nil {
			t.Error(err)
		}
		cfParsed2, err := c.loadResolvedFile(testDockerComposeYml)
		if err != nil {
			t.Error(err)
		}
		if cfParsed1 != cfParsed2 {
			t.Fail()
		}
	})
}

func TestConfigLoaderLoadResolvedFile_OpenFileError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlIOError)
		if err == nil {
			t.Fail()
		}
	})
}

func TestConfigLoaderLoadResolvedFile_VersionError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlInvalidVersion)
		if err == nil {
			t.Fail()
		}
	})
}

func TestConfigLoaderLoadResolvedFile_InterpolationError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlInterpolationIssue)
		if err == nil {
			t.Fail()
		}
	})
}

func TestConfigLoaderLoadResolvedFile_DecodeError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlDecodeIssue)
		if err == nil {
			t.Fail()
		}
	})
}

func TestNew_ExtendsCycle(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlExtendsCycle,
		})
		if err == nil {
			t.Fail()
		}
	})
}
func TestNew_Success(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{})
		if err != nil {
			t.Error(err)
		}
	})
}
func TestNew_ExtendsSuccess(t *testing.T) {
	withMockFS(func() {
		c, err := New([]string{testDockerComposeYmlExtends})
		if err != nil {
			t.Error(err)
		} else {
			assertServiceMapsEqual(t, c.Services, map[string]*Service{
				"service1": &Service{
					Environment: map[string]string{
						"KEY1": "VALUE1",
						"KEY2": "VALUE2",
					},
				},
				"service2": &Service{
					Environment: map[string]string{
						"KEY2": "VALUE2",
					},
				},
				"service3": &Service{
					Environment: map[string]string{
						"KEY2": "VALUE2",
					},
				},
			})
		}
	})
}

func TestNew_StandardFileError(t *testing.T) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystemStandardFileError
	_, err := New([]string{})
	if err == nil {
		t.Fail()
	}
}

func TestGetVersion_Default(t *testing.T) {
	m := genericMap{}
	v, err := getVersion(m)
	if err != nil {
		t.Error(err)
	}
	if v == nil || !v.Equal(v1) {
		t.Fail()
	}
}

func TestGetVersion_FormatError(t *testing.T) {
	m := genericMap{
		"version": "",
	}
	_, err := getVersion(m)
	if err == nil {
		t.Fail()
	}
}

func TestGetVersion_TypeError(t *testing.T) {
	m := genericMap{
		"version": 0,
	}
	_, err := getVersion(m)
	if err == nil {
		t.Fail()
	}
}

func TestGetVersion_Success(t *testing.T) {
	m := genericMap{
		"version": "1.0",
	}
	v, err := getVersion(m)
	if err != nil {
		t.Error(err)
	}
	if v == nil || !v.Equal(v1) {
		t.Fail()
	}
}

func TestComposeFileParsedServiceClearRecStack_Success(t *testing.T) {
	s := &composeFileParsedService{}
	s.recStack = true
	s.clearRecStack()
	if s.recStack {
		t.Fail()
	}
}

func TestLoadFileError_Success(t *testing.T) {
	err := loadFileError("some file", fmt.Errorf("an error occured"))
	if err == nil {
		t.Fail()
	}
}

func TestParseComposeFileService_InvalidPortsError(t *testing.T) {
	c := newTestConfigLoader(nil)
	cfService := &composeFileService{
		Ports: []port{
			{
				Value: "asdf",
			},
		},
	}
	_, err := c.parseComposeFileService(cfService)
	if err == nil {
		t.Fail()
	}
}

func TestParseComposeFileService_InvalidHealthcheckError(t *testing.T) {
	c := newTestConfigLoader(nil)
	cfService := &composeFileService{
		Healthcheck: &ServiceHealthcheck{
			Timeout: new(string),
		},
	}
	*cfService.Healthcheck.Timeout = "henkie"
	_, err := c.parseComposeFileService(cfService)
	if err == nil {
		t.Fail()
	}
}

func TestParseComposeFileService_InvalidEnvironmentError(t *testing.T) {
	c := newTestConfigLoader(nil)
	cfService := &composeFileService{
		Environment: environment{
			Values: []environmentNameValuePair{
				{
					Name: "",
				},
			},
		},
	}
	_, err := c.parseComposeFileService(cfService)
	if err == nil {
		t.Fail()
	}
}

func TestGetXProperties_NotGenericMap(t *testing.T) {
	v := getXProperties("")
	if v != nil {
		t.Fail()
	}
}

func TestGetXProperties_Success(t *testing.T) {
	key1 := "x-key1"
	val1 := "val1"
	key2 := "x-key2"
	val2 := "val2"
	v := getXProperties(genericMap{
		key1: val1,
		key2: val2,
	})
	if len(v) != 2 {
		t.Fail()
	}
	if v[key1] != val1 {
		t.Fail()
	}
	if v[key2] != val2 {
		t.Fail()
	}
}
