package config

import (
	"reflect"
	"testing"
)

const testValue = "val1"

func mapValueGetter(m map[string]string) ValueGetter {
	return func(name string) (string, bool) {
		value, found := m[name]
		return value, found
	}
}

func TestInterpolate_Simple1(t *testing.T) {
	m := map[string]string{
		"VAR1": testValue,
	}
	str, err := Interpolate("$VAR1", mapValueGetter(m), true)
	if err != nil || str != testValue {
		t.Fatal(str, err)
	}
}

func TestInterpolate_Simple2(t *testing.T) {
	m := map[string]string{
		"VAR1": testValue,
	}
	str, err := Interpolate("$VAR1 ", mapValueGetter(m), true)
	if err != nil || str != "val1 " {
		t.Fatal(str, err)
	}
}

func TestInterpolate_EOF(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("$", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolate_DefaultValue(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$VAR1", mapValueGetter(m), true)
	if err != nil || str != "" {
		t.Fatal(str, err)
	}
}

func TestInterpolate_DollarSign1(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$", mapValueGetter(m), true)
	if err != nil || str != "$" {
		t.Fatal(str, err)
	}
}
func TestInterpolate_DollarSign2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$ ", mapValueGetter(m), true)
	if err != nil || str != "$ " {
		t.Fatal(str, err)
	}
}

func TestInterpolate_UnexpectedRune(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("$[", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolate_BracesSimple(t *testing.T) {
	m := map[string]string{
		"VAR1": testValue,
	}
	str, err := Interpolate("${VAR1}", mapValueGetter(m), true)
	if err != nil || str != testValue {
		t.Fatal(str, err)
	}
}

func TestInterpolate_BracesEOF(t *testing.T) {
	m := map[string]string{
		"VAR1": testValue,
	}
	_, err := Interpolate("${VAR1", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolate_BracesDefaultValue1(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("${VAR1}", mapValueGetter(m), true)
	if err != nil || str != "" {
		t.Fatal(err)
	}
}

func TestInterpolate_BracesDefaultValue2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("${VAR1-val1}", mapValueGetter(m), true)
	if err != nil || str != testValue {
		t.Fatal(err)
	}
}

func TestInterpolate_BracesDefaultValue3(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	str, err := Interpolate("${VAR1:-val1}", mapValueGetter(m), true)
	if err != nil || str != testValue {
		t.Fatal(err)
	}
}
func TestInterpolate_BracesError1(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("${VAR1?errorMsg1}", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolate_BracesError2(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	_, err := Interpolate("${VAR1:?errorMsg1}", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolate_BracesError3(t *testing.T) {
	expected := "value"
	m := map[string]string{
		"VAR1": expected,
	}
	actual, err := Interpolate("${VAR1:?errorMsg1}", mapValueGetter(m), true)
	if err != nil {
		t.Error(err)
	} else if actual != expected {
		t.Fail()
	}
}

func TestInterpolate_BracesInvalidDelimiter(t *testing.T) {
	m := map[string]string{
		"VAR:ABLE": testValue,
	}
	str, err := Interpolate("${VAR:ABLE}", mapValueGetter(m), true)
	if err != nil || str != testValue {
		t.Fatal(str, err)
	}
}

func TestInterpolate_RecursiveSlice(t *testing.T) {
	m := map[string]string{}
	c := &configInterpolator{
		valueGetter: mapValueGetter(m),
		version:     v2_1,
	}
	input := []string{
		"$$",
	}
	output := c.interpolateRecursive(input, path{})
	if !reflect.DeepEqual(output, []string{"$"}) {
		t.Fail()
	}
}
func TestInterpolate_RecursiveMap(t *testing.T) {
	m := map[string]string{}
	c := &configInterpolator{
		valueGetter: mapValueGetter(m),
		version:     v2_1,
	}
	input := genericMap{
		"key": "value",
	}
	outputRaw := c.interpolateRecursive(input, path{})
	output, ok := outputRaw.(genericMap)
	if !ok || len(output) != 1 || output["key"] != "value" || len(c.errorList) > 0 {
		t.Fail()
	}
}

func TestInterpolate_NestedErrors(t *testing.T) {
	m := map[string]string{}
	c := &configInterpolator{
		valueGetter: mapValueGetter(m),
		version:     v2_1,
	}
	input := genericMap{
		"key": "$",
	}
	c.interpolateRecursive(input, path{})
	if len(c.errorList) == 0 {
		t.Fail()
	}
}

func TestInterpolateConfig_V1(t *testing.T) {
	m := map[string]string{}
	config := map[interface{}]interface{}{
		"service1": "$$",
	}
	err := InterpolateConfig(config, mapValueGetter(m), v1)
	if len(config) != 1 || config["service1"] != "$" {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestInterpolateConfig_V1Error(t *testing.T) {
	m := map[string]string{}
	config := map[interface{}]interface{}{
		"service1": "$",
	}
	err := InterpolateConfig(config, mapValueGetter(m), v1)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateConfig_V3(t *testing.T) {
	m := map[string]string{}
	config := genericMap{
		"secrets": genericMap{
			"secret1": "$$",
		},
	}
	err := InterpolateConfig(config, mapValueGetter(m), v3_3)
	if !reflect.DeepEqual(config, genericMap{
		"secrets": genericMap{
			"secret1": "$",
		},
	}) {
		t.Log(config)
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}
