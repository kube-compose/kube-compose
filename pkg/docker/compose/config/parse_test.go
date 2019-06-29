package config

import (
	"reflect"
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/util"
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
	src := "testPortDecodeSuccessString"
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
		File:    util.NewString(file),
		Service: service,
	}
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
	expected := environmentValue{
		Int64Value: new(int64),
	}
	*expected.Int64Value = int64(src)
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentValuesEqual(&dst, &expected) {
		t.Fail()
	}
}

func TestEnvironmentValueDecode_FractionalFloat64Success(t *testing.T) {
	src := 123.5
	var dst environmentValue
	expected := environmentValue{
		FloatValue: new(float64),
	}
	*expected.FloatValue = src
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentValuesEqual(&dst, &expected) {
		t.Fail()
	}
}

func TestEnvironmentValueDecode_StringSuccess(t *testing.T) {
	src := "environmentValueStringSuccess"
	var dst environmentValue
	expected := environmentValue{
		StringValue: util.NewString(src),
	}
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentValuesEqual(&dst, &expected) {
		t.Logf("actual  : %+v\n", dst)
		t.Logf("expected: %+v\n", expected)
		t.Fail()
	}
}

func TestEnvironmentValueDecode_NilSuccess(t *testing.T) {
	var dst environmentValue
	err := mapdecode.Decode(&dst, nil)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentValuesEqual(&dst, &environmentValue{}) {
		t.Fail()
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

func TestEnvironmentDecode_MapSuccess(t *testing.T) {
	src := map[string]interface{}{
		"VAR1": "VAL1",
		"VAR2": 0,
		"VAR3": 123.5,
		"VAR4": nil,
	}
	var dst environment
	exp := environment{
		Values: []environmentNameValuePair{
			{
				Name: "VAR1",
				Value: &environmentValue{
					StringValue: util.NewString("VAL1"),
				},
			},
			{
				Name: "VAR2",
				Value: &environmentValue{
					Int64Value: new(int64),
				},
			},
			{
				Name: "VAR3",
				Value: &environmentValue{
					FloatValue: new(float64),
				},
			},
			{
				Name:  "VAR4",
				Value: &environmentValue{},
			},
		},
	}
	*exp.Values[2].Value.FloatValue = 123.5
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentsEqual(dst, exp) {
		t.Logf("env1: %+v\n", dst)
		t.Logf("env2: %+v\n", exp)
		t.Fail()
	}
}

func TestEnvironmentDecode_SliceSuccess(t *testing.T) {
	src := []string{
		"VAR5=VAL5",
		"VAR6",
	}
	var dst environment
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !areEnvironmentsEqual(dst, environment{
		Values: []environmentNameValuePair{
			{
				Name: "VAR5",
				Value: &environmentValue{
					StringValue: util.NewString("VAL5"),
				},
			},
			{
				Name: "VAR6",
			},
		},
	}) {
		t.Fail()
	}
}

func areEnvironmentsEqual(env1, env2 environment) bool {
	n := len(env1.Values)
	if n != len(env2.Values) {
		return false
	}
	if hasDuplicateNames(env1) || hasDuplicateNames(env2) {
		panic("env1 or env2 has duplicate namees")
	}
	for i := 0; i < n; i++ {
		pair1 := &env1.Values[i]
		found := false
		for j := 0; j < n; j++ {
			pair2 := &env2.Values[j]
			if pair1.Name == pair2.Name && areEnvironmentValuesEqual(pair1.Value, pair2.Value) {
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

func hasDuplicateNames(env environment) bool {
	hasName := map[string]bool{}
	for _, pair := range env.Values {
		if _, ok := hasName[pair.Name]; ok {
			return true
		}
		hasName[pair.Name] = true
	}
	return false
}

func areEnvironmentValuesEqual(v1, v2 *environmentValue) bool {
	if v1 == nil || v2 == nil {
		return v1 == v2
	}
	if !util.FloatPointersPointToSameValue(v1.FloatValue, v2.FloatValue) {
		return false
	}
	return reflect.DeepEqual(v1.Int64Value, v2.Int64Value) &&
		reflect.DeepEqual(v1.StringValue, v2.StringValue)
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
	if !reflect.DeepEqual(dst.Values, []string{HealthcheckCommandShell, src}) {
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

func TestServiceVolumeDecode_Success(t *testing.T) {
	src := "aa:bb:cc"
	var dst ServiceVolume
	err := mapdecode.Decode(&dst, src)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(dst, ServiceVolume{
		Short: &PathMapping{
			ContainerPath: "bb",
			HasHostPath:   true,
			HasMode:       true,
			HostPath:      "aa",
			Mode:          "cc",
		},
	}) {
		t.Fail()
	}
}

func TestServiceVolumeDecode_Error(t *testing.T) {
	src := 0
	var dst ServiceVolume
	err := mapdecode.Decode(&dst, src)
	if err == nil {
		t.Fail()
	}
}
