package util

import (
	"fmt"
	"reflect"
	"testing"
)

type testHasSubexpNames struct {
	subexpNames []string
}

func (t *testHasSubexpNames) SubexpNames() []string {
	return t.subexpNames
}

func TestBuildRegexpMatchMap(t *testing.T) {
	subexpNamesContainer := &testHasSubexpNames{
		subexpNames: []string{
			"",
			"group1",
		},
	}
	m := BuildRegexpMatchMap(subexpNamesContainer, []string{
		"",
		"group1Match",
	})
	if !reflect.DeepEqual(m, map[string]string{
		"group1": "group1Match",
	}) {
		t.Fail()
	}
}

type testCloser struct {
	closed bool
	err    error
}

func (c *testCloser) Close() error {
	c.closed = true
	return c.err
}

func TestCloseAndLogError(t *testing.T) {
	c := testCloser{
		err: fmt.Errorf(""),
	}
	CloseAndLogError(&c)
	if !c.closed {
		t.Fail()
	}
}

func TestEscapeName_Success(t *testing.T) {
	r := EscapeName("\x00\x390a\x7B!")
	// Each character that is not [a-z0-8] is replaced by a three-letter sequence 9[a-z0-9]{2}, i.e.:
	// "\x00" => "9aa"
	// "\x39" => "9bv"
	// "0" 	  => "0"
	// "a"    => "a"
	// "\x7B" => "9dp"
	// "!" 	  => "9a7"
	if r != "9aa9bv0a9dp9a7" {
		t.Fail()
	}
}

func TestEscapeName_Success2(t *testing.T) {
	r := EscapeName("--a-z089--")
	if r != "9bj-a-z089bv-9bj" {
		t.Fail()
	}
}

func TestTryParseInt64_Error(t *testing.T) {
	uid := TryParseInt64("asdf")
	if uid != nil {
		t.Fail()
	}
}
func TestTryParseInt64_Success(t *testing.T) {
	uid := TryParseInt64("234")
	if uid == nil || *uid != 234 {
		t.Fail()
	}
}

func TestFormatTable(t *testing.T) {
	output := FormatTable([][]string{
		{"NAME", "VALUE"},
		{"Test", "-1"},
	})
	if output != "NAME  VALUE\nTest  -1\n" {
		t.Fail()
	}
}
func TestUnescapeName_Success(t *testing.T) {
	r, err := UnescapeName("9aa9bv0a9dp9a7")
	if r != "\x00\x390a\x7B!" || err != nil {
		t.Fail()
	}
}

func TestUnescapeName_Success2(t *testing.T) {
	r, err := UnescapeName("9bj-a-z089bv-9bj")
	if r != "--a-z089--" || err != nil {
		t.Fail()
	}
}

func TestUnescapeName_Error(t *testing.T) {
	_, err := UnescapeName("9zz")
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByte_Error1(t *testing.T) {
	_, err := unescapeByte("9\x00a", 0)
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByte_Error2(t *testing.T) {
	_, err := unescapeByte("9a\x00", 0)
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByte_Error3(t *testing.T) {
	_, err := unescapeByte("", 0)
	if err == nil {
		t.Fail()
	}
}

func TestFloatPointersPointToSameValue_OneNil(t *testing.T) {
	fp1 := new(float64)
	if FloatPointersPointToSameValue(fp1, nil) {
		t.Fail()
	}
}
