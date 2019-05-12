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
