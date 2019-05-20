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
