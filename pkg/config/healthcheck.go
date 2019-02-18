package config

import (
	"fmt"
	"time"
)

const (
	healthcheckCommandShell = "CMD-SHELL"
)

type Healthcheck struct {
	Interval time.Duration
	IsShell bool
	StartPeriod time.Duration
	Retries uint
	Test []string
	Timeout time.Duration
}

func parseHealthcheck (healthcheckYAML *healthcheckCompose2_1) (*Healthcheck, bool, error) {
	if healthcheckYAML == nil {
		return nil, false, nil
	}
	
	if healthcheckYAML.Disable {
		return nil, true, nil
	}

	// Parse Test
	test := healthcheckYAML.GetTest()
	if len(test) == 0 {
		return nil, false, fmt.Errorf("Field \"test\" of Healthcheck must not be an empty array")
	}
	healthcheck := &Healthcheck{}
	switch test[0] {
	case "NONE":
	case "CMD":
	case healthcheckCommandShell:
		healthcheck.IsShell = true
	default:
		return nil, false, fmt.Errorf("Field \"test\" of Healthcheck must have a first element that is one of \"NONE\", \"CMD\" and \"%s\"\n", healthcheckCommandShell)
	}
	if test[0] == "NONE" {
		return nil, true, nil
	}
	if len(test) == 1 {
		return nil, false, fmt.Errorf("Field \"test\" of Healthcheck must have size at least 2 of its first element is not \"NONE\"")
	}
	healthcheck.Test = test[1:]

	// We do not set StartPeriod because it is unsupported in docker-compose 2.1 and
	// should therefore  be treated as 0.
	
	// time.ParseDuration supports a superset of duration compared to docker-compose:
	// https://golang.org/pkg/time/#Duration
	// https://docs.docker.com/compose/compose-file/compose-file-v2/#specifying-durations

	interval, err := time.ParseDuration(healthcheckYAML.Interval)
	if err != nil {
		return nil, false, err
	}
	if interval <= 0 {
		return nil, false, fmt.Errorf("Field \"interval\" of Healthcheck must not be negative")
	}
	healthcheck.Interval = interval

	timeout, err := time.ParseDuration(healthcheckYAML.Interval)
	if err != nil {
		return nil, false, err
	}
	if interval <= 0 {
		return nil, false, fmt.Errorf("Field \"timeout\" of Healthcheck must not be negative")
	}
	healthcheck.Timeout = timeout

	healthcheck.Retries = healthcheckYAML.Retries
	
	return healthcheck, false, nil
}