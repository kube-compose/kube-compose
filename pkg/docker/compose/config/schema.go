package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	"github.com/uber-go/mapdecode"
)

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

type ServiceHealthcheck struct {
	Disable  bool            `mapdecode:"disable"`
	Interval *string         `mapdecode:"interval"`
	Retries  *uint           `mapdecode:"retries"`
	Test     HealthcheckTest `mapdecode:"test"`
	Timeout  *string         `mapdecode:"timeout"`
	// start_period is only available in docker-compose 2.3 or higher
}

func (h *ServiceHealthcheck) GetTest() []string {
	return h.Test.Values
}

type dependsOn struct {
	Values map[string]ServiceHealthiness
}

func (t *dependsOn) Decode(into mapdecode.Into) error {
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
	Value *environmentValue
}

// TODO https://github.com/jbrekelmans/kube-compose/issues/40 check whether handling of large numbers is consistent with docker-compose
// See https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json#L418
type environmentValue struct {
	FloatValue  *float64
	Int64Value  *int64
	StringValue *string
}

func (v *environmentValue) Decode(into mapdecode.Into) error {
	var f float64
	err := into(&f)
	if err == nil {
		if -9223372036854775000.0 <= f && f <= 9223372036854775000.0 && math.Floor(f) == f {
			v.Int64Value = new(int64)
			*v.Int64Value = int64(f)
			return nil
		}
		v.FloatValue = new(float64)
		*v.FloatValue = f
		return nil
	}
	var s string
	err = into(&s)
	if err == nil {
		v.StringValue = util.NewString(s)
	}
	return err
}

type environment struct {
	Values []environmentNameValuePair
}

func (t *environment) Decode(into mapdecode.Into) error {
	var intoMap map[string]environmentValue
	err := into(&intoMap)
	if err == nil {
		i := 0
		t.Values = make([]environmentNameValuePair, len(intoMap))
		for name, value := range intoMap {
			t.Values[i].Name = name
			valueCopy := new(environmentValue)
			*valueCopy = value
			t.Values[i].Value = valueCopy
			i++
		}
		return nil
	}
	var intoSlice []string
	err = into(&intoSlice)
	if err == nil {
		t.Values = make([]environmentNameValuePair, len(intoSlice))
		for i, nameValuePair := range intoSlice {
			j := strings.IndexRune(nameValuePair, '=')
			if j < 0 {
				t.Values[i].Name = nameValuePair
			} else {
				t.Values[i].Name = nameValuePair[:j]
				stringValue := nameValuePair[j+1:]
				t.Values[i].Value = &environmentValue{
					StringValue: &stringValue,
				}
			}
		}
	}
	return err
}

type extendsHelper struct {
	File    *string `mapdecode:"file"`
	Service string  `mapdecode:"service"`
}

type extends struct {
	File    *string
	Service string
}

// Used by mapdecode package
func (e *extends) Decode(into mapdecode.Into) error {
	var serviceName string
	err := into(&serviceName)
	if err == nil {
		e.Service = serviceName
		return nil
	}
	var eHelper extendsHelper
	err = into(&eHelper)
	if err != nil {
		return err
	}
	e.File = eHelper.File
	e.Service = eHelper.Service
	return nil
}

type port struct {
	Value string
}

func (p *port) Decode(into mapdecode.Into) error {
	var int64Val int64
	err := into(&int64Val)
	if err == nil {
		p.Value = strconv.FormatInt(int64Val, 10)
		return nil
	}
	strVal := ""
	err = into(&strVal)
	p.Value = strVal
	return err
}

type composeFileService struct {
	Build *struct {
		Context    string `mapdecode:"context"`
		Dockerfile string `mapdecode:"dockerfile"`
	} `mapdecode:"build"`
	Command     stringOrStringSlice  `mapdecode:"command"`
	DependsOn   *dependsOn           `mapdecode:"depends_on"`
	Entrypoint  *stringOrStringSlice `mapdecode:"entrypoint"`
	Environment environment          `mapdecode:"environment"`
	Extends     *extends             `mapdecode:"extends"`
	Healthcheck *ServiceHealthcheck  `mapdecode:"healthcheck"`
	Image       string               `mapdecode:"image"`
	Ports       []port               `mapdecode:"ports"`
	User        *string              `mapdecode:"user"`
	Volumes     []string             `mapdecode:"volumes"`
	WorkingDir  string               `mapdecode:"working_dir"`
	Restart     string               `mapdecode:"restart"`
}

type composeFile struct {
	Services map[string]*composeFileService `mapdecode:"services"`
	Volumes  map[string]interface{}         `mapdecode:"volumes"`
}
