package config

import (
	"testing"
)

func newTestConfig() *Config {
	serviceA := &Service{
		name: "a",
	}
	serviceB := &Service{
		name: "b",
	}
	serviceC := &Service{
		name: "c",
	}
	serviceD := &Service{
		name: "d",
	}
	serviceA.DependsOn = map[*Service]ServiceHealthiness{
		serviceB: ServiceHealthy,
	}
	serviceB.DependsOn = map[*Service]ServiceHealthiness{
		serviceC: ServiceHealthy,
		serviceD: ServiceHealthy,
	}
	cfg := &Config{
		CanonicalComposeFile: CanonicalComposeFile{
			Services: map[string]*Service{
				serviceA.Name(): serviceA,
				serviceB.Name(): serviceB,
				serviceC.Name(): serviceC,
				serviceD.Name(): serviceD,
			},
			Version: v2_1,
		},
	}
	return cfg
}

func TestSetFilter(t *testing.T) {
	cfg := newTestConfig()

	// Since a depends on b, and b depends on c and d, we expect the result to contain all 4 apps.
	err := cfg.SetFilter([]string{"a"})
	if err != nil {
		t.Fail()
	}
	_, resultContainsAppA := cfg.filter["a"]
	_, resultContainsAppB := cfg.filter["b"]
	_, resultContainsAppC := cfg.filter["c"]
	_, resultContainsAppD := cfg.filter["d"]
	if !resultContainsAppA || !resultContainsAppB || !resultContainsAppC || !resultContainsAppD {
		t.Fail()
	}
}
