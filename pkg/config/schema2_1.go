package config

import (
	"fmt"
	"strings"
)

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json

type ServiceHealthiness int

const (
	ServiceStarted ServiceHealthiness = 0
	ServiceHealthy ServiceHealthiness = 1
)

func (s *ServiceHealthiness) String() string {
	switch *s {
	case ServiceHealthy:
		return "serviceHealthy"
	case ServiceStarted:
		return "serviceStarted"
	}
	return ""
}

type stringOrStringSlice struct {
	Values []string
}

func (t *stringOrStringSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.Values)
	if err != nil {
		var str string
		err = unmarshal(&str)
		if err != nil {
			return err
		}
		t.Values = []string{str}
	}
	return nil
}

type HealthcheckTest struct {
	Values []string
}

func (t *HealthcheckTest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.Values)
	if err != nil {
		var str string
		err = unmarshal(&str)
		if err != nil {
			return err
		}
		t.Values = []string{
			HealthcheckCommandShell,
			str,
		}
	}
	return nil
}

type ServiceHealthcheck2_1 struct {
	Disable  bool            `yaml:"disable"`
	Interval *string         `yaml:"interval"`
	Retries  *uint           `yaml:"retries"`
	Test     HealthcheckTest `yaml:"test"`
	Timeout  *string         `yaml:"timeout"`
	// start_period is only available in docker-compose 2.3 or higher
}

func (h *ServiceHealthcheck2_1) GetTest() []string {
	return h.Test.Values
}

type dependsOn2_1 struct {
	Values map[string]ServiceHealthiness
}

func (t *dependsOn2_1) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

type environmentNameValuePair struct {
	Name  string
	Value *string
}

type environment2_1 struct {
	KeysMayHaveVariableSubstitutions bool
	Values                           []environmentNameValuePair
}

func (t *environment2_1) UnmarshalYAML(unmarshal func(interface{}) error) error {
	t.KeysMayHaveVariableSubstitutions = false
	var strMap map[string]string
	err := unmarshal(&strMap)
	if err == nil {
		i := 0
		t.Values = make([]environmentNameValuePair, len(strMap))
		for name, value := range strMap {
			t.Values[i].Name = name
			t.Values[i].Value = &value
			i++
		}
		return nil
	}
	t.KeysMayHaveVariableSubstitutions = true
	var strSlice []string
	err = unmarshal(&strSlice)
	if err == nil {
		t.Values = make([]environmentNameValuePair, len(strSlice))
		for i, nameValuePair := range strSlice {
			j := strings.IndexRune(nameValuePair, '=')
			if j < 0 {
				t.Values[i].Name = nameValuePair
				t.Values[i].Value = nil
			} else {
				t.Values[i].Name = nameValuePair[:j]
				value := nameValuePair[j+1:]
				t.Values[i].Value = &value
			}
		}
	}
	return err
}

type serviceYAML2_1 struct {
	Build struct {
		Context    string `yaml:"context"`
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"build"`
	DependsOn   dependsOn2_1           `yaml:"depends_on"`
	Entrypoint  stringOrStringSlice    `yaml:"entrypoint"`
	Environment environment2_1         `yaml:"environment"`
	Healthcheck *ServiceHealthcheck2_1 `yaml:"healthcheck"`
	Image       string                 `yaml:"image"`
	Ports       []string               `yaml:"ports"`
	Volumes     []string               `yaml:"volumes"`
	WorkingDir  string                 `yaml:"working_dir"`
}

type composeYAML2_1 struct {
	Services map[string]serviceYAML2_1 `yaml:"services"`
	Volumes  map[string]interface{}    `yaml:"volumes"`
}
