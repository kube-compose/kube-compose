package config

import (
	"fmt"
	"reflect"
	"testing"
)

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
	m := map[string]string{
		name1: value1,
	}
	c := &configLoader{
		environmentGetter: mapValueGetter(m),
	}
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
	m := map[string]string{}
	c := &configLoader{
		environmentGetter: mapValueGetter(m),
	}
	_, err := c.parseEnvironment(input)
	if err == nil {
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
	m := map[string]string{}
	c := &configLoader{
		environmentGetter: mapValueGetter(m),
	}
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
	m := map[string]string{}
	c := &configLoader{
		environmentGetter: mapValueGetter(m),
	}
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

/*

func (c *configLoader) parseComposeFileService(cfService *composeFileService) (*composeFileParsedService, error) {
	service := &Service{
		Entrypoint: cfService.Entrypoint.Values,
		Image:      cfService.Image,
		User:       cfService.User,
		WorkingDir: cfService.WorkingDir,
		Restart:    cfService.Restart,
	}
	composeFileParsedService := &composeFileParsedService{
		service: service,
	}
	if cfService.DependsOn != nil {
		composeFileParsedService.dependsOn = cfService.DependsOn.Values
	}
	ports, err := parsePorts(cfService.Ports)
	if err != nil {
		return nil, err
	}
	service.Ports = ports

	healthcheck, healthcheckDisabled, err := ParseHealthcheck(cfService.Healthcheck)
	if err != nil {
		return nil, err
	}
	service.Healthcheck = healthcheck
	service.HealthcheckDisabled = healthcheckDisabled

	environment, err := c.parseEnvironment(cfService.Environment.Values)
	if err != nil {
		return nil, err
	}
	service.Environment = environment

	return composeFileParsedService, nil
}
*/
