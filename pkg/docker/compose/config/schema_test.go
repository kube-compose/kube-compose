package config

import (
	"reflect"
	"testing"

	"github.com/uber-go/mapdecode"
)

func TestPortDecode_SuccessInt(t *testing.T) {
	src := 9223372036854775807
	var dst port
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.Value != "9223372036854775807" {
		t.Error(dst.Value)
	}
}

func TestPortDecode_SuccessString(t *testing.T) {
	src := "9223372036854775806"
	var dst port
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.Value != src {
		t.Error(dst.Value)
	}
}

func TestExtendsDecode_SuccessString(t *testing.T) {
	src := "my-service"
	var dst extends
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.File != nil || dst.Service != src {
		t.Error(dst)
	}
}

func TestExtendsDecode_Error(t *testing.T) {
	src := 1234
	var dst extends
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

func TestExtendsDecode_SuccessComplex(t *testing.T) {
	file := "another-file"
	service := "my-service-in-another-file"
	src := &extendsHelper{
		File:    new(string),
		Service: service,
	}
	*src.File = file
	var dst extends
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.Service != service || dst.File == nil || *dst.File != file {
		t.Error(dst)
	}
}

func TestEnvironmentValueDecode_SmallIntegralFloat64Success(t *testing.T) {
	src := 123.0
	var dst environmentValue
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.FloatValue != nil || dst.StringValue != nil || dst.Int64Value == nil || *dst.Int64Value != 123 {
		t.Error(dst)
	}
}

func TestEnvironmentValueDecode_FractionalFloat64Success(t *testing.T) {
	src := 123.5
	var dst environmentValue
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.FloatValue == nil || dst.StringValue != nil || dst.Int64Value != nil || *dst.FloatValue != 123.5 {
		t.Error(dst)
	}
}

func TestEnvironmentValueDecode_StringSuccess(t *testing.T) {
	src := "environmentValueStringSuccess"
	var dst environmentValue
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if dst.FloatValue != nil || dst.StringValue == nil || dst.Int64Value != nil || *dst.StringValue != src {
		t.Error(dst)
	}
}

func TestEnvironmentValueDecode_NilSuccess(t *testing.T) {
	var dst environmentValue
	err := mapdecode.Decode(&dst, nil)
	if err != nil {
		t.Error(err)
	}
	if dst.FloatValue != nil || dst.StringValue != nil || dst.Int64Value != nil {
		t.Error(dst)
	}
}

func TestEnvironmentValueDecode_Error(t *testing.T) {
	src := []string{}
	var dst environmentValue
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

// TODO https://github.com/jbrekelmans/kube-compose/issues/64 ignoring cyclomatic complexity errors
// nolint
func TestEnvironmentDecode_MapSuccess(t *testing.T) {
	src := map[string]interface{}{
		"VAR1": "VAL1",
		"VAR2": 1234,
		"VAR3": 123.4,
		"VAR4": nil,
	}
	var dst environment
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	flags := 0
	for _, pair := range dst.Values {
		value := pair.Value
		switch pair.Name {
		case "VAR1":
			if value.FloatValue != nil || value.Int64Value != nil || value.StringValue == nil || *value.StringValue != "VAL1" {
				t.Fail()
			}
			flags |= 1
		case "VAR2":
			if value.FloatValue != nil || value.Int64Value == nil || value.StringValue != nil || *value.Int64Value != 1234 {
				t.Fail()
			}
			flags |= 2
		case "VAR3":
			if value.FloatValue == nil || value.Int64Value != nil || value.StringValue != nil || *value.FloatValue != 123.4 {
				t.Fail()
			}
			flags |= 4
		case "VAR4":
			if value.FloatValue != nil || value.Int64Value != nil || value.StringValue != nil {
				t.Fail()
			}
			flags |= 8
		default:
			t.Fail()
		}
	}
	if flags != 15 {
		t.Fail()
	}
}

// TODO https://github.com/jbrekelmans/kube-compose/issues/64 ignoring cyclomatic complexity errors
// nolint
func TestEnvironmentDecode_SliceSuccess(t *testing.T) {
	src := []string{
		"VAR5=VAL5",
		"VAR6",
	}
	var dst environment
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	flags := 0
	for _, pair := range dst.Values {
		value := pair.Value
		switch pair.Name {
		case "VAR5":
			if value.FloatValue != nil || value.Int64Value != nil || value.StringValue == nil || *value.StringValue != "VAL5" {
				t.Fail()
			}
			flags |= 1
		case "VAR6":
			if value != nil {
				t.Fail()
			}
			flags |= 2
		default:
			t.Fail()
		}
	}
	if flags != 3 {
		t.Fail()
	}
}

func TestEnvironmentDecode_Error(t *testing.T) {
	src := "asdf"
	var dst environment
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

func TestDependsOnDecode_MapSuccess(t *testing.T) {
	src := map[string]map[string]string{
		"service-bla-1": {
			"condition": "service_healthy",
		},
		"service-bla-2": {
			"condition": "service_started",
		},
	}
	var dst dependsOn
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, map[string]ServiceHealthiness{
		"service-bla-1": ServiceHealthy,
		"service-bla-2": ServiceStarted,
	}) {
		t.Error(dst)
	}
}

func TestDependsOnDecode_MapInvalidCondition(t *testing.T) {
	src := map[string]map[string]string{
		"service-bla-6": {
			"condition": "asdf",
		},
	}
	var dst dependsOn
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}
func TestDependsOnDecode_SliceSuccess(t *testing.T) {
	src := []string{
		"service-bla-3",
	}
	var dst dependsOn
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, map[string]ServiceHealthiness{
		"service-bla-3": ServiceStarted,
	}) {
		t.Error(dst)
	}
}

func TestDependsOnDecode_SliceDuplicatesError(t *testing.T) {
	src := []string{
		"service-bla-4",
		"service-bla-4",
	}
	var dst dependsOn
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

func TestDependsOnDecode_Error(t *testing.T) {
	src := "henk"
	var dst dependsOn
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

func TestHealthcheckTestDecode_SliceSuccess(t *testing.T) {
	src := []string{
		HealthcheckCommandShell,
		"echo 'Hello World 234!'",
	}
	var dst HealthcheckTest
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, src) {
		t.Error(dst)
	}
}

func TestHealthcheckTestDecode_StringSuccess(t *testing.T) {
	src := "echo 'Hello World 23!'"
	var dst HealthcheckTest
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, append([]string{HealthcheckCommandShell, src})) {
		t.Error(dst)
	}
}

func TestHealthcheckTestDecode_Error(t *testing.T) {
	src := 15989
	var dst HealthcheckTest
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}

func TestStringOrStringSliceDecode_StringSuccess(t *testing.T) {
	src := "stringOrStringSlice1"
	var dst stringOrStringSlice
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, []string{src}) {
		t.Error(dst)
	}
}
func TestStringOrStringSliceDecode_StringSliceSuccess(t *testing.T) {
	src := []string{"stringOrStringSlice2"}
	var dst stringOrStringSlice
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dst.Values, src) {
		t.Error(dst)
	}
}

func TestStringOrStringSliceDecode_Error(t *testing.T) {
	var src int
	var dst stringOrStringSlice
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}
