package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/pkg/errors"
)

const testDockerComposeYml = "/docker-compose.yaml"
const testDockerComposeYmlIOError = "/docker-compose.io-error.yaml"
const testDockerComposeYmlInvalidVersion = "/docker-compose.invalid-version.yml"
const testDockerComposeYmlInterpolationIssue = "/docker-compose.interpolation-issue.yml"
const testDockerComposeYmlDecodeIssue = "/docker-compose.decode-issue.yml"
const testDockerComposeYmlExtends = "/docker-compose.extends.yml"
const testDockerComposeYmlExtendsCycle = "/docker-compose.extends-cycle.yml"
const testDockerComposeYmlExtendsIOError = "/docker-compose.extends-io-error.yml"
const testDockerComposeYmlExtendsDoesNotExist = "/docker-compose.extends-does-not-exist.yml"
const testDockerComposeYmlExtendsDoesNotExistFile = "/docker-compose.extends-does-not-exist-file.yml"
const testDockerComposeYmlExtendsInvalidDependsOn = "/docker-compose.extends-invalid-depends-on.yml"
const testDockerComposeYmlDependsOnDoesNotExist = "/docker-compose.depends-on-does-not-exist.yml"
const testDockerComposeYmlDependsOnCycle1 = "/docker-compose.depends-on-cycle-1.yml"
const testDockerComposeYmlDependsOnCycle2 = "/docker-compose.depends-on-cycle-2.yml"
const testDockerComposeYmlDependsOn = "/docker-compose.depends-on.yml"
const testDockerComposeYmlInvalidHealthcheck = "/docker-compose.invalid-healthcheck.yml"

var mockFS = fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
	testDockerComposeYml: {
		Content: []byte(`testservice:
  entrypoint: []
  command: ["bash", "-c", "echo 'Hello World!'"]
  image: ubuntu:latest
  volumes:
  - "aa:bb:cc"
`),
	},
	testDockerComposeYmlIOError: {
		Error: errors.New("unknown error 1"),
	},
	testDockerComposeYmlInvalidVersion: {
		Content: []byte("version: ''"),
	},
	testDockerComposeYmlInterpolationIssue: {
		Content: []byte(`version: '2.3'
services:
  testservice:
    image: '$'
`),
	},
	testDockerComposeYmlDecodeIssue: {
		Content: []byte(`version: '2.3'
services:
  testservice:
    environment: 3
`),
	},
	testDockerComposeYmlExtends: {
		Content: []byte(`version: '2.3'
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
      file: '` + testDockerComposeYml[1:] + `'
      service: testservice
  service3:
    extends:
      file: '` + testDockerComposeYml[1:] + `'
      service: testservice
`),
	},
	testDockerComposeYmlExtendsCycle: {
		Content: []byte(`version: '2.3'
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
	testDockerComposeYmlExtendsIOError: {
		Content: []byte(`version: '2.3'
services:
  service1:
    environment:
      KEY2: VALUE2
    extends:
      file: '` + testDockerComposeYmlIOError + `'
      service: testservice
`),
	},
	testDockerComposeYmlExtendsDoesNotExist: {
		Content: []byte(`version: '2.3'
services:
  service1:
    extends:
      service: service2
`),
	},
	testDockerComposeYmlExtendsDoesNotExistFile: {
		Content: []byte(`version: '2.3'
services:
  service1:
    extends:
      file: '` + testDockerComposeYml + `'
      service: service2
`),
	},
	testDockerComposeYmlExtendsInvalidDependsOn: {
		Content: []byte(`version: '2.3'
services:
  service1:
    extends:
      service: service2
  service2:
    depends_on:
    - service1
`),
	},
	testDockerComposeYmlDependsOnDoesNotExist: {
		Content: []byte(`version: '2.3'
services:
  service1:
    depends_on:
    - service2
`),
	},
	testDockerComposeYmlDependsOnCycle1: {
		Content: []byte(`version: '2.3'
services:
  service1:
    depends_on:
    - service2
  service2:
    depends_on:
    - service1
`),
	},
	testDockerComposeYmlDependsOnCycle2: {
		Content: []byte(`version: '2.3'
services:
  service1:
	command: []
`),
	},
	testDockerComposeYmlDependsOn: {
		Content: []byte(`version: '2.3'
services:
  service1:
    depends_on:
    - service2
  service2:
    depends_on:
      service3:
        condition: service_healthy
  service3: {}
`),
	},
	testDockerComposeYmlInvalidHealthcheck: {
		Content: []byte(`version: '2.3'
services:
  service1:
	healthcheck:
      timeout: henkie
`),
	},
})

var mockFileSystemStandardFileError = fs.NewInMemoryUnixFileSystem(map[string]fs.InMemoryFile{
	"/docker-compose.yml": {
		Error: errors.New("unknown error 2"),
	},
})

func withMockFS(cb func()) {
	original := fs.OS
	defer func() {
		fs.OS = original
	}()
	fs.OS = mockFS
	cb()
}

func newTestConfigLoader(env map[string]string) *configLoader {
	c := &configLoader{
		environmentGetter:     mapValueGetter(env),
		loadResolvedFileCache: map[string]*loadResolvedFileCacheItem{},
	}
	return c
}

func Test_ConfigLoader_ParseEnvironment_Success(t *testing.T) {
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
func Test_ConfigLoader_ParseEnvironment_InvalidName(t *testing.T) {
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

func Test_ConfigLoader_LoadFile_Success(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		dcFile, err := c.loadFile(testDockerComposeYml)
		if err != nil {
			t.Error(err)
		} else {
			if !dcFile.version.Equal(v1) {
				t.Fail()
			}
			if dcFile.xProperties != nil {
				t.Fail()
			}
			expected := map[string]*serviceInternal{
				"testservice": {
					Command: &stringOrStringSlice{
						Values: []string{"bash", "-c", "echo 'Hello World!'"},
					},
					Entrypoint: &stringOrStringSlice{
						Values: []string{},
					},
					Image: util.NewString("ubuntu:latest"),
					Volumes: []ServiceVolume{
						{
							Short: &PathMapping{
								ContainerPath: "bb",
								HasHostPath:   true,
								HasMode:       true,
								HostPath:      "aa",
								Mode:          "cc",
							},
						},
					},
				},
			}
			assertServiceInternalMapsEqual(t, dcFile.Services, expected)
		}
	})
}

func assertServiceInternalMapsEqual(t *testing.T, m1, m2 map[string]*serviceInternal) {
	if (m1 == nil) != (m2 == nil) {
		t.Fail()
		return
	}
	if m1 == nil {
		return
	}
	if len(m1) != len(m2) {
		t.Fail()
		return
	}
	for name, s1 := range m1 {
		s2 := m2[name]
		if s2 == nil {
			t.Fail()
			return
		}
		assertServiceInternalEqual(t, s1, s2)
	}
}

func areStringOrStringSlicesEqual(s1, s2 *stringOrStringSlice) bool {
	if s1 == nil {
		return s2 == nil
	}
	if s2 == nil {
		return false
	}
	return areStringSlicesEqual(s1.Values, s2.Values)
}

func assertServiceInternalEqual(t *testing.T, s1, s2 *serviceInternal) {
	if !areStringOrStringSlicesEqual(s1.Command, s2.Command) {
		t.Fail()
		return
	}
	if !areDependsOnEqual(s1.DependsOn, s2.DependsOn) {
		t.Fail()
		return
	}
	if !areStringOrStringSlicesEqual(s1.Entrypoint, s2.Entrypoint) {
		t.Fail()
		return
	}
	if !areStringMapsEqual(s1.environmentParsed, s2.environmentParsed) {
		t.Fail()
		return
	}
	if !areExtendsEqual(s1.Extends, s2.Extends) {
		t.Fail()
		return
	}
	if s1.Healthcheck != nil || s2.Healthcheck != nil {
		t.Logf("comparing healthchecks is not supported")
		t.Fail()
		return
	}
	assertServiceInternalEqualContinued(t, s1, s2)
}

func areExtendsEqual(e1, e2 *extends) bool {
	if (e1 == nil) != (e2 == nil) {
		return false
	}
	if e1 == nil {
		return true
	}
	if e1.Service != e2.Service {
		return false
	}
	return areStringPointersEqual(e1.File, e2.File)
}

func assertServiceInternalEqualContinued(t *testing.T, s1, s2 *serviceInternal) {
	if !areStringPointersEqual(s1.Image, s2.Image) {
		t.Fail()
		return
	}
	if !arePortsEqual(s1.portsParsed, s2.portsParsed) {
		t.Fail()
		return
	}
	if !areBoolPointersEqual(s1.Privileged, s2.Privileged) {
		t.Fail()
		return
	}
	if !areStringPointersEqual(s1.Restart, s2.Restart) {
		t.Fail()
		return
	}
	if !areStringPointersEqual(s1.User, s2.User) {
		t.Fail()
		return
	}
	if !areServiceVolumesEqual(s1.Volumes, s2.Volumes) {
		t.Fail()
		return
	}
	if !areStringPointersEqual(s1.WorkingDir, s2.WorkingDir) {
		t.Fail()
		return
	}
}

func areBoolPointersEqual(b1, b2 *bool) bool {
	if (b1 == nil) != (b2 == nil) {
		return false
	}
	return b1 == nil || *b1 == *b2
}

func areStringPointersEqual(s1, s2 *string) bool {
	if (s1 == nil) != (s2 == nil) {
		return false
	}
	return s1 == nil || *s1 == *s2
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
			assertAreDependsOnEqual(t, name, services1, services2)
		}
	}
}

func assertAreDependsOnEqual(t *testing.T, name string, services1, services2 map[string]*Service) {
	dependsOn1 := services1[name].DependsOn
	dependsOn2 := services2[name].DependsOn
	if (dependsOn1 == nil) != (dependsOn2 == nil) {
		t.Fail()
		return
	}
	if dependsOn2 == nil {
		return
	}
	if len(dependsOn1) != len(dependsOn2) {
		t.Fail()
		return
	}
	names := reverseMap(services1)
	for service1, healthiness1 := range dependsOn1 {
		name, ok := names[service1]
		if !ok {
			panic("dependsOn refers to a service that is not in services")
		}
		service2 := services2[name]
		if service2 == nil {
			t.Fail()
			return
		}
		healthiness2, ok := dependsOn2[service2]
		if !ok || healthiness1 != healthiness2 {
			t.Fail()
			return
		}
	}
}

func reverseMap(services map[string]*Service) map[*Service]string {
	ret := map[*Service]string{}
	for name, service := range services {
		if _, ok := ret[service]; ok {
			panic("services is invalid")
		}
		ret[service] = name
	}
	return ret
}

func areDependsOnEqual(m1, m2 *dependsOn) bool {
	if m1 == nil {
		return m2 == nil
	}
	if m2 == nil {
		return false
	}
	if len(m1.Values) != len(m2.Values) {
		return false
	}
	return isDependsOnMapSubsetOf(m1.Values, m2.Values) && isDependsOnMapSubsetOf(m2.Values, m1.Values)
}

func isDependsOnMapSubsetOf(m1, m2 map[string]ServiceHealthiness) bool {
	for name, healthiness1 := range m1 {
		healthiness2, ok := m2[name]
		if !ok || healthiness1 != healthiness2 {
			return false
		}
	}
	return true
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
	if !arePortsEqual(service1.Ports, service2.Ports) {
		t.Logf("ports1: %+v\n", service1.Ports)
		t.Logf("ports2: %+v\n", service2.Ports)
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
	if !areStringMapsEqual(service1.Environment, service2.Environment) {
		t.Logf("env1: %+v\n", service1.Environment)
		t.Logf("env2: %+v\n", service2.Environment)
		t.Fail()
	}
	if !areStringSlicesEqual(service1.Entrypoint, service2.Entrypoint) {
		t.Logf("entrypoint1: %+v\n", service1.Entrypoint)
		t.Logf("entrypoint2: %+v\n", service2.Entrypoint)
		t.Fail()
	}
	if !areStringSlicesEqual(service1.Command, service2.Command) {
		t.Logf("command1: %+v\n", service1.Command)
		t.Logf("command2: %+v\n", service2.Command)
		t.Fail()
	}
	if !areServiceVolumesEqual(service1.Volumes, service2.Volumes) {
		t.Logf("volumes1: %+v\n", service1.Volumes)
		t.Logf("volumes2: %+v\n", service2.Volumes)
		t.Fail()
	}
}

func areServiceVolumesEqual(volumes1, volumes2 []ServiceVolume) bool {
	if volumes1 == nil {
		return volumes2 == nil
	}
	if volumes2 == nil {
		return false
	}
	n := len(volumes1)
	if n != len(volumes2) {
		return false
	}
	for i := 0; i < n; i++ {
		if !arePathMappingsEqual(volumes1[i].Short, volumes2[i].Short) {
			return false
		}
	}
	return true
}

func arePathMappingsEqual(pm1, pm2 *PathMapping) bool {
	if pm1 == nil {
		return pm2 == nil
	}
	return pm2 != nil && *pm1 == *pm2
}

func areStringMapsEqual(m1, m2 map[string]string) bool {
	if m1 == nil {
		return m2 == nil
	}
	if m2 == nil {
		return false
	}
	if len(m1) != len(m2) {
		return false
	}
	for key, value1 := range m1 {
		value2, ok := m2[key]
		if !ok || value1 != value2 {
			return false
		}
	}
	return true
}

func areStringSlicesEqual(slice1, slice2 []string) bool {
	if slice1 == nil {
		return slice2 == nil
	}
	if slice2 == nil {
		return false
	}
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

func Test_ConfigLoader_LoadFile_Error(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadFile(testDockerComposeYmlIOError)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_ConfigLoader_LoadResolvedFile_Caching(t *testing.T) {
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

func Test_ConfigLoader_LoadResolvedFile_OpenFileError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlIOError)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_ConfigLoader_LoadResolvedFile_VersionError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlInvalidVersion)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_ConfigLoader_LoadResolvedFile_InterpolationError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlInterpolationIssue)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_ConfigLoader_LoadResolvedFile_DecodeError(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadResolvedFile(testDockerComposeYmlDecodeIssue)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_New_DependsOnDoesNotExist(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlDependsOnDoesNotExist,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_DependsOnCycle1(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlDependsOnCycle1,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_DependsOnCycle2(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlDependsOnCycle1,
			testDockerComposeYmlDependsOnCycle2,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_DependsOnSuccess(t *testing.T) {
	withMockFS(func() {
		c, err := New([]string{
			testDockerComposeYmlDependsOn,
		})
		if err != nil {
			t.Error(err)
		} else {
			service1 := &Service{}
			service2 := &Service{}
			service3 := &Service{}
			service1.DependsOn = map[*Service]ServiceHealthiness{
				service2: ServiceStarted,
			}
			service2.DependsOn = map[*Service]ServiceHealthiness{
				service3: ServiceHealthy,
			}
			assertServiceMapsEqual(t, c.Services, map[string]*Service{
				"service1": service1,
				"service2": service2,
				"service3": service3,
			})
		}
	})
}

func Test_New_InvalidHealthcheckError(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlInvalidHealthcheck,
		})
		if err == nil {
			t.Fail()
		}
	})
}

func Test_New_IOError(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlIOError,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_MultipleFiles(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYml,
			testDockerComposeYmlExtends,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_ExtendsCycle(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{
			testDockerComposeYmlExtendsCycle,
		})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_ExtendsSuccess(t *testing.T) {
	withMockFS(func() {
		c, err := New([]string{testDockerComposeYmlExtends})
		if err != nil {
			t.Error(err)
		} else {
			assertServiceMapsEqual(t, c.Services, map[string]*Service{
				"service1": {
					Command:    []string{"bash", "-c", "echo 'Hello World!'"},
					Entrypoint: []string{},
					Environment: map[string]string{
						"KEY1": "VALUE1",
						"KEY2": "VALUE2",
					},
					Image: "ubuntu:latest",
					Volumes: []ServiceVolume{
						{
							Short: &PathMapping{
								ContainerPath: "bb",
								HasHostPath:   true,
								HasMode:       true,
								HostPath:      "aa",
								Mode:          "cc",
							},
						},
					},
				},
				"service2": {
					Command:    []string{"bash", "-c", "echo 'Hello World!'"},
					Entrypoint: []string{},
					Environment: map[string]string{
						"KEY2": "VALUE2",
					},
					Image: "ubuntu:latest",
					Volumes: []ServiceVolume{
						{
							Short: &PathMapping{
								ContainerPath: "bb",
								HasHostPath:   true,
								HasMode:       true,
								HostPath:      "aa",
								Mode:          "cc",
							},
						},
					},
				},
				"service3": {
					Command:    []string{"bash", "-c", "echo 'Hello World!'"},
					Entrypoint: []string{},
					Image:      "ubuntu:latest",
					Volumes: []ServiceVolume{
						{
							Short: &PathMapping{
								ContainerPath: "bb",
								HasHostPath:   true,
								HasMode:       true,
								HostPath:      "aa",
								Mode:          "cc",
							},
						},
					},
				},
			})
		}
	})
}

func Test_New_ExtendsIOError(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{testDockerComposeYmlExtendsIOError})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_ExtendsDoesNotExist(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{testDockerComposeYmlExtendsDoesNotExist})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_ExtendsDoesNotExistFile(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{testDockerComposeYmlExtendsDoesNotExistFile})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}
func Test_New_ExtendsInvalidDependsOn(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{testDockerComposeYmlExtendsInvalidDependsOn})
		if err == nil {
			t.Fail()
		} else {
			t.Log(err)
		}
	})
}

func Test_New_Success(t *testing.T) {
	withMockFS(func() {
		_, err := New([]string{})
		if err != nil {
			t.Error(err)
		}
	})
}
func Test_New_StandardFileError(t *testing.T) {
	orig := fs.OS
	defer func() {
		fs.OS = orig
	}()
	fs.OS = mockFileSystemStandardFileError
	_, err := New([]string{})
	if err == nil {
		t.Fail()
	}
}

func Test_GetVersion_Default(t *testing.T) {
	m := genericMap{}
	v, err := getVersion(m)
	if err != nil {
		t.Error(err)
	}
	if v == nil || !v.Equal(v1) {
		t.Fail()
	}
}

func Test_GetVersion_FormatError(t *testing.T) {
	m := genericMap{
		"version": "",
	}
	_, err := getVersion(m)
	if err == nil {
		t.Fail()
	}
}

func Test_GetVersion_TypeError(t *testing.T) {
	m := genericMap{
		"version": 0,
	}
	_, err := getVersion(m)
	if err == nil {
		t.Fail()
	}
}

func Test_GetVersion_Success(t *testing.T) {
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

func Test_ServiceInternal_ClearRecStack_Success(t *testing.T) {
	s := &serviceInternal{}
	s.recStack = true
	s.clearRecStack()
	if s.recStack {
		t.Fail()
	}
}

func Test_LoadFileError_Success(t *testing.T) {
	err := loadFileError("some file", fmt.Errorf("an error occurred"))
	if err == nil {
		t.Fail()
	}
}

func Test_ConfigLoader_ParseDockerComposeFileService_InvalidPortsError(t *testing.T) {
	c := newTestConfigLoader(nil)

	dcFile := &dockerComposeFile{
		Services: map[string]*serviceInternal{
			"service1": {
				Ports: []port{
					{
						Value: "asdf",
					},
				},
			},
		},
	}
	s := dcFile.Services["service1"]
	err := c.parseDockerComposeFileService(dcFile, s)
	if err == nil {
		t.Fail()
	}
}

func Test_ConfigLoader_ParseDockerComposeFile_InvalidEnvironmentError(t *testing.T) {
	c := newTestConfigLoader(nil)
	dcFile := &dockerComposeFile{
		Services: map[string]*serviceInternal{
			"service1": {
				Environment: &environment{
					Values: []environmentNameValuePair{
						{
							Name: "",
						},
					},
				},
			},
		},
	}
	err := c.parseDockerComposeFile(dcFile)
	if err == nil {
		t.Fail()
	} else {
		t.Log(err)
	}
}

func Test_GetXProperties_NotGenericMap(t *testing.T) {
	v := getXProperties("")
	if v != nil {
		t.Fail()
	}
}

func Test_GetXProperties_Success(t *testing.T) {
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
