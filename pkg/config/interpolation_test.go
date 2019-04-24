package config

import (
	"testing"
)

func mapValueGetter(m map[string]string) ValueGetter {
	return func(name string) (string, bool) {
		value, found := m[name]
		return value, found
	}
}

func TestInterpolateSimple1(t *testing.T) {
	m := map[string]string{
		"VAR1": "val1",
	}
	str, err := Interpolate("$VAR1", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fatal(str, err)
	}
}

func TestInterpolateSimple2(t *testing.T) {
	m := map[string]string{
		"VAR1": "val1",
	}
	str, err := Interpolate("$VAR1 ", mapValueGetter(m), true)
	if err != nil || str != "val1 " {
		t.Fatal(str, err)
	}
}

func TestInterpolateEOF(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("$", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateDefaultValue(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$VAR1", mapValueGetter(m), true)
	if err != nil || str != "" {
		t.Fatal(str, err)
	}
}

func TestInterpolateDollarSign1(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$", mapValueGetter(m), true)
	if err != nil || str != "$" {
		t.Fatal(str, err)
	}
}
func TestInterpolateDollarSign2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$ ", mapValueGetter(m), true)
	if err != nil || str != "$ " {
		t.Fatal(str, err)
	}
}

func TestInterpolateUnexpectedRune(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("$[", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateBracesSimple(t *testing.T) {
	m := map[string]string{
		"VAR1": "val1",
	}
	str, err := Interpolate("${VAR1}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fatal(str, err)
	}
}

func TestInterpolateBracesEOF(t *testing.T) {
	m := map[string]string{
		"VAR1": "val1",
	}
	_, err := Interpolate("${VAR1", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateBracesDefaultValue1(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("${VAR1}", mapValueGetter(m), true)
	if err != nil || str != "" {
		t.Fatal(err)
	}
}

func TestInterpolateBracesDefaultValue2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("${VAR1-val1}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fatal(err)
	}
}

func TestInterpolateBracesDefaultValue3(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	str, err := Interpolate("${VAR1:-val1}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fatal(err)
	}
}
func TestInterpolateBracesError1(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("${VAR1?errorMsg1}", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateBracesError2(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	_, err := Interpolate("${VAR1:?errorMsg1}", mapValueGetter(m), true)
	if err == nil {
		t.Fail()
	}
}

func TestInterpolateBracesInvalidDelimiter(t *testing.T) {
	m := map[string]string{
		"VAR:ABLE": "val1",
	}
	str, err := Interpolate("${VAR:ABLE}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fatal(str, err)
	}
}

func TestInterpolateRecursiveSlice(t *testing.T) {
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
