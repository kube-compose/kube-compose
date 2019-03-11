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
		t.Fail()
	}
}

func TestInterpolateSimple2(t *testing.T) {
	m := map[string]string{
		"VAR1": "val1",
	}
	str, err := Interpolate("$VAR1 ", mapValueGetter(m), true)
	if err != nil || str != "val1 " {
		t.Fail()
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
		t.Fail()
	}
}

func TestInterpolateDollarSign1(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$", mapValueGetter(m), true)
	if err != nil || str != "$" {
		t.Fail()
	}
}
func TestInterpolateDollarSign2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("$$ ", mapValueGetter(m), true)
	if err != nil || str != "$ " {
		t.Fail()
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
		t.Fail()
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
		t.Fail()
	}
}

func TestInterpolateBracesDefaultValue2(t *testing.T) {
	m := map[string]string{}
	str, err := Interpolate("${VAR1-val1}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fail()
	}
}

func TestInterpolateBracesDefaultValue3(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	str, err := Interpolate("${VAR1:-val1}", mapValueGetter(m), true)
	if err != nil || str != "val1" {
		t.Fail()
	}
}
func TestInterpolateBracesError1(t *testing.T) {
	m := map[string]string{}
	_, err := Interpolate("${VAR1?errorMsg1}", mapValueGetter(m), true)
	if err != nil {
		t.Fail()
	}
}

func TestInterpolateBracesError2(t *testing.T) {
	m := map[string]string{
		"VAR1": "",
	}
	_, err := Interpolate("${VAR1:?errorMsg1}", mapValueGetter(m), true)
	if err != nil {
		t.Fail()
	}
}

func TestInterpolateBracesInvalidDelimiter(t *testing.T) {
	m := map[string]string{
		"VAR:ABLE": "val1",
	}
	str, err := Interpolate("${VAR:ABLE}", mapValueGetter(m), true)
	if err == nil || str != "val1" {
		t.Fail()
	}
}
