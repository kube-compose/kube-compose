package config

import "testing"

func TestParsePortBindings_InternalMinTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65536", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_InternalMaxTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65535-65536", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_ExternalMinTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65536:8000", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_ExternalMaxTooLarge(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("65535-65536:8000-8001", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_RandomlyAvailable(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8001:8000", portBindings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortBindings_RangeLengthMismatch(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8002:8000-8001", portBindings)
	if err == nil {
		t.Fail()
	}
}

func TestParsePortBindings_SuccessWithExternal(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("127.0.0.1:8000-8001:8000-8001/udp", portBindings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsePortBindings_SuccessWithoutExternal(t *testing.T) {
	portBindings := []PortBinding{}
	_, err := parsePortBindings("8000-8001", portBindings)
	if err != nil {
		t.Fatal(err)
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
