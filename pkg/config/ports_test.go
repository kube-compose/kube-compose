package config

import "testing"

func TestParsePortBindingsInternalMinTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65536", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindingsInternalMaxTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65535-65536", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindingsExternalMinTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65536:8000", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindingsExternalMaxTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65535-65536:8000-8001", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindingsRandomlyAvailable(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8001:8000", portBindings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortBindingsRangeLengthMismatch(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8002:8000-8001", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindingsSuccessWithExternal(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("myhostname:8000-8001:8000-8001/udp", portBindings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortBindingsSuccessWithoutExternal(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8001", portBindings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortUintSuccess(t *testing.T) {
	_, err := parsePortUint("8000")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortUintInvalidFormat(t *testing.T) {
	_, err := parsePortUint("-1")
	if err == nil {
		t.Fail()
	}
}

func TestParsePortUintTooLarge(t *testing.T) {
	_, err := parsePortUint("65536")
	if err == nil {
		t.Fail()
	}
}