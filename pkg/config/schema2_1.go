package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uber-go/mapdecode"
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

func (t *stringOrStringSlice) Decode(into mapdecode.Into) error {
	err := into(&t.Values)
	if err != nil {
		var str string
		err = into(&str)
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

func (t *HealthcheckTest) Decode(into mapdecode.Into) error {
	err := into(&t.Values)
	if err != nil {
		var str string
		err = into(&str)
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
	Disable  bool            `mapdecode:"disable"`
	Interval *string         `mapdecode:"interval"`
	Retries  *uint           `mapdecode:"retries"`
	Test     HealthcheckTest `mapdecode:"test"`
	Timeout  *string         `mapdecode:"timeout"`
	// start_period is only available in docker-compose 2.3 or higher
}

func (h *ServiceHealthcheck2_1) GetTest() []string {
	return h.Test.Values
}

type dependsOn2_1 struct {
	Values map[string]ServiceHealthiness
}

func (t *dependsOn2_1) Decode(into mapdecode.Into) error {
	var strMap map[string]struct {
		Condition string `mapdecode:"condition"`
	}
	err := into(&strMap)
	if err != nil {
		var services []string
		err = into(&services)
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
	Values []environmentNameValuePair
}

func (t *environment2_1) Decode(into mapdecode.Into) error {
	var strMap map[string]string
	err := into(&strMap)
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
	var strSlice []string
	err = into(&strSlice)
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

type port struct {
	Value string
}

func (p *port) Decode(into mapdecode.Into) error {
	intVal := 0
	err := into(&intVal)
	if err == nil {
		p.Value = strconv.Itoa(intVal)
		return nil
	}
	strVal := ""
	err = into(&strVal)
	p.Value = strVal
	return err
}

type service2_1 struct {
	Build struct {
		Context    string `mapdecode:"context"`
		Dockerfile string `mapdecode:"dockerfile"`
	} `mapdecode:"build"`
	DependsOn   dependsOn2_1           `mapdecode:"depends_on"`
	Entrypoint  stringOrStringSlice    `mapdecode:"entrypoint"`
	Environment environment2_1         `mapdecode:"environment"`
	Healthcheck *ServiceHealthcheck2_1 `mapdecode:"healthcheck"`
	Image       string                 `mapdecode:"image"`
	Ports       []port                 `mapdecode:"ports"`
	Volumes     []string               `mapdecode:"volumes"`
	WorkingDir  string                 `mapdecode:"working_dir"`
}

type composeFile2_1 struct {
	Services map[string]service2_1  `mapdecode:"services"`
	Volumes  map[string]interface{} `mapdecode:"volumes"`
}
