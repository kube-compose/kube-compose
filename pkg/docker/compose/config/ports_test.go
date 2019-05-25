package config

import (
	"reflect"
	"testing"
)

func TestParsePortBindings_InternalMinTooLarge(t *testing.T) {
	_, err := parsePortBindings("65536", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_InternalMaxTooLarge(t *testing.T) {
	_, err := parsePortBindings("65535-65536", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_ExternalMinTooLarge(t *testing.T) {
	_, err := parsePortBindings("65536:8000", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_ExternalMaxTooLarge(t *testing.T) {
	_, err := parsePortBindings("65535-65536:8000-8001", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_RandomlyAvailable(t *testing.T) {
	expected := []PortBinding{
		{
			Internal:    8000,
			ExternalMin: 8000,
			ExternalMax: 8001,
			Protocol:    "tcp",
		},
	}
	actual, err := parsePortBindings("8000-8001:8000", nil)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Logf("ports1: %+v\n", actual)
		t.Logf("ports2: %+v\n", expected)
		t.Fail()
	}
}

func TestParsePortBindings_RangeLengthMismatch(t *testing.T) {
	_, err := parsePortBindings("8000-8002:8000-8001", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_SuccessWithExternal1(t *testing.T) {
	expected := []PortBinding{
		{
			Internal:    8000,
			ExternalMin: 8000,
			ExternalMax: 8000,
			Protocol:    "udp",
			Host:        "127.0.0.1",
		},
		{
			Internal:    8001,
			ExternalMin: 8001,
			ExternalMax: 8001,
			Protocol:    "udp",
			Host:        "127.0.0.1",
		},
	}
	actual, err := parsePortBindings("127.0.0.1:8000-8001:8000-8001/udp", nil)
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Logf("ports1: %+v\n", actual)
		t.Logf("ports2: %+v\n", expected)
		t.Fail()
	}
}
func TestParsePortBindings_SuccessWithExternal2(t *testing.T) {
	expected := []PortBinding{
		{
			Internal:    8000,
			ExternalMin: 8000,
			ExternalMax: 8000,
			Protocol:    "udp",
			Host:        "127.0.0.1",
		},
	}
	actual, err := parsePortBindings("127.0.0.1:8000:8000/udp", nil)
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Logf("ports1: %+v\n", actual)
		t.Logf("ports2: %+v\n", expected)
		t.Fail()
	}
}

func TestParsePortBindings_SuccessWithoutExternal(t *testing.T) {
	expected := []PortBinding{
		{
			Internal:    8000,
			ExternalMin: -1,
			ExternalMax: -1,
			Protocol:    "tcp",
		},
		{
			Internal:    8001,
			ExternalMin: -1,
			ExternalMax: -1,
			Protocol:    "tcp",
		},
	}
	actual, err := parsePortBindings("8000-8001", nil)
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Logf("ports1: %+v\n", actual)
		t.Logf("ports2: %+v\n", expected)
		t.Fail()
	}
}
func TestParsePortBindings_Error(t *testing.T) {
	_, err := parsePortBindings("!", nil)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortUint_Success(t *testing.T) {
	_, err := parsePortUint("8000")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortUint_InvalidFormat(t *testing.T) {
	_, err := parsePortUint("-1")
	if err == nil {
		t.Fail()
	}
}

func TestParsePortUint_TooLarge(t *testing.T) {
	_, err := parsePortUint("65536")
	if err == nil {
		t.Fail()
	}
}

func TestParsePorts_Error(t *testing.T) {
	p := port{
		Value: "!",
	}
	_, err := parsePorts([]port{
		p,
	})
	if err == nil {
		t.Fail()
	}
}

func TestParsePorts_Success(t *testing.T) {
	_, _ = parsePorts([]port{})
}
