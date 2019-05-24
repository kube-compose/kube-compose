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
			if !reflect.DeepEqual(cfParsed.xProperties, map[string]interface{}{}) {
				t.Fail()
			}
			assertComposeFileParsedServicesEqual(t, cfParsed.services, map[string]*composeFileParsedService{
				"audit-service": {
					service: &Service{
						Image: "ubuntu:latest",
					},
				},
			})
		}
	})
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

func TestConfigLoaderLoadStandardFile_Success(t *testing.T) {
	withMockFS(func() {
		c := newTestConfigLoader(nil)
		_, err := c.loadStandardFile()
		if err != nil {
			t.Error(err)
		}
	})
}
func TestConfigLoaderLoadStandardFile_Error(t *testing.T) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystemStandardFileError
	c := newTestConfigLoader(nil)
	_, err := c.loadStandardFile()
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

func assertComposeFileParsedServicesEqual(t *testing.T, services1, services2 map[string]*composeFileParsedService) {
	if len(services1) != len(services2) {
		t.Fail()
	}
	for name, cfServiceParsed1 := range services1 {
		cfServiceParsed2 := services2[name]
		if cfServiceParsed2 == nil {
			t.Fail()
		} else {
			// Comparing the depends on is not implemented in this assert function.
			if cfServiceParsed1.dependsOn != nil || cfServiceParsed2.dependsOn != nil {
				t.Fail()
			}
			if cfServiceParsed1.extends != cfServiceParsed2.extends {
				t.Fail()
			}
			if cfServiceParsed1.service == nil {
				if cfServiceParsed2.service != nil {
					t.Fail()
				}
			} else if !reflect.DeepEqual(*cfServiceParsed1.service, *cfServiceParsed2.service) {
				t.Fail()
			}
		}
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

func TestParseComposeFile_InvalidServiceNameError(t *testing.T) {
	c := newTestConfigLoader(nil)
	cfParsed := &composeFileParsed{}
	cf := &composeFile{
		Services: map[string]*composeFileService{
			"!!": {},
		},
	}
	err := c.parseComposeFile(cf, cfParsed)
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
	if !reflect.DeepEqual(v, map[string]interface{}{
		key1: val1,
		key2: val2,
	}) {
		t.Fail()
	}
}
