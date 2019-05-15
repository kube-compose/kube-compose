package config

import (
	"testing"
)

func TestMerge_Success(t *testing.T) {
	serviceA := &Service{
		Environment: map[string]string{
			"VAR1": "val1a",
		},
	}
	serviceB := &Service{
		Environment: map[string]string{
			"VAR1": "val1b",
			"VAR2": "val2b",
		},
	}
	merge(serviceA, serviceB)

	expectedEnv := map[string]string{
		"VAR1": "val1a",
		"VAR2": "val2b",
	}
	actualEnv := serviceA.Environment
	for key, value := range expectedEnv {
		if actualEnv[key] != value {
			t.Fail()
		}
	}
	if len(actualEnv) != len(expectedEnv) {
		t.Fail()
	}
}

func TestMerge_Basic(t *testing.T) {
	serviceA := &Service{
		Name:        "a",
		Environment: map[string]string{"a": "b"},
		Ports:       []PortBinding{{80, 80, 80, "tcp", ""}},
	}

	serviceB := &Service{
		Name:        "b",
		Environment: map[string]string{"b": "c"},
		Ports:       []PortBinding{{8000, 8000, 8000, "tcp", ""}},
	}

	expected := &Service{
		Name:        "a",
		Environment: map[string]string{"a": "b", "b": "c"},
		Ports:       []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}},
	}

	merge(serviceA, serviceB)
	if !reflect.DeepEqual(serviceA, expected) {
		t.Fail()
	}
}
