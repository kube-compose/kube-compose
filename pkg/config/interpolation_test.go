package config

import (
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
	outputRaw := c.interpolateRecursive(input, path{})
	if output, ok := outputRaw.([]string); ok {
		if len(output) != 1 || output[0] != "$" {
			t.Fail()
		}
		return
	}
	t.Fail()
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
	if !ok || &input != &output || len(c.errorList) > 0 {
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
