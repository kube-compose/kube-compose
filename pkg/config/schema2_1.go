package config

import (
	"fmt"
)

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json

type ServiceHealthiness int

const (
	ServiceStarted ServiceHealthiness = 0
	ServiceHealthy ServiceHealthiness = 1
)

type healthcheckTest struct {
	Values []string
}

func (t *healthcheckTest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.Values)
	if err != nil {
		var str string
		err = unmarshal(&str)
		if err != nil {
			return err
		}
		t.Values = []string{
			healthcheckCommandShell,
			str,
		}
	}
	return nil
}

type healthcheckCompose2_1 struct {
	Disable  bool   `yaml:"disable"`
	Interval string `yaml:"interval"`
	Retries  uint   `yaml:"retries"`
	// start_period is only available in docker-compose 2.3 or higher
	Test    healthcheckTest `yaml:"test"`
	Timeout string          `yaml:"timeout"`
}

func (h *healthcheckCompose2_1) GetTest() []string {
	return h.Test.Values
}

type dependsOnYAML2_1 struct {
	Values map[string]ServiceHealthiness
}

func (t *dependsOnYAML2_1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strMap map[string]struct {
		Condition string `yaml:"condition"`
	}
	err := unmarshal(&strMap)
	if err != nil {
		var services []string
		err = unmarshal(&services)
		if err != nil {
			return err
		}
		t.Values = map[string]ServiceHealthiness{}
		for _, service := range services {
			_, ok := t.Values[service]
			if ok {
				return fmt.Errorf("depends_on list cannot contain duplicate values")
			}
			t.Values[service] = ServiceStarted
		}
	} else {
		n := len(strMap)
		t.Values = make(map[string]ServiceHealthiness, n)
		for service, obj := range strMap {
			switch obj.Condition {
			case "service_healthy":
				t.Values[service] = ServiceHealthy
			case "service_started":
				t.Values[service] = ServiceStarted
			default:
				return fmt.Errorf("depends_on map contains an entry with an invalid condition: %s", obj.Condition)
			}
		}
	}
	return nil
}

type serviceYAML2_1 struct {
	Build struct {
		Context    string `yaml:"context"`
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"build"`
	DependsOn   dependsOnYAML2_1       `yaml:"depends_on"`
	Environment map[string]string      `yaml:"environment"`
	Healthcheck *healthcheckCompose2_1 `yaml:"healthcheck"`
	Image       string                 `yaml:"image"`
	Ports       []string               `yaml:"ports"`
	Volumes     []string               `yaml:"volumes"`
	WorkingDir  string                 `yaml:"working_dir"`
}

type composeYAML2_1 struct {
	Services map[string]serviceYAML2_1 `yaml:"services"`
	Volumes  map[string]interface{}    `yaml:"volumes"`
}
