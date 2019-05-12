package config

import (
	"testing"
)

func newTestConfig() *Config {
	serviceA := &Service{
		ServiceName: "a",
	}
	serviceB := &Service{
		ServiceName: "b",
	}
	serviceC := &Service{
		ServiceName: "c",
	}
	serviceD := &Service{
		ServiceName: "d",
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
				serviceA.ServiceName: serviceA,
				serviceB.ServiceName: serviceB,
				serviceC.ServiceName: serviceC,
				serviceD.ServiceName: serviceD,
			},
			Version: v2_1,
		},
	}
	return cfg
}

func TestSetFilter(t *testing.T) {
	cfg := newTestConfig()

	// Since a depends on b, and b depends on c and d, we expect the result to contain all 4 apps.
	cfg.SetFilter([]string{"a"})

	_, resultContainsAppA := cfg.filter["a"]
	_, resultContainsAppB := cfg.filter["b"]
	_, resultContainsAppC := cfg.filter["c"]
	_, resultContainsAppD := cfg.filter["d"]
	if !resultContainsAppA || !resultContainsAppB || !resultContainsAppC || !resultContainsAppD {
		t.Fail()
	}
}
