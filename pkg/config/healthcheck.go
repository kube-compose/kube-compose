package config

import (
	"fmt"
	"time"
)

const (
	HealthcheckCommandShell    = "CMD-SHELL"
	HealthcheckCommandCmd      = "CMD"
	HealthcheckCommandNone     = "NONE"
	HealthcheckDefaultInterval = 30 * time.Second
	HealthcheckDefaultTimeout  = 30 * time.Second
	HealthcheckDefaultRetries  = 3
)

type Healthcheck struct {
	Interval    time.Duration
	IsShell     bool
	StartPeriod time.Duration
	Retries     uint
	Test        []string
	Timeout     time.Duration
}

<<<<<<< HEAD
// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
=======
>>>>>>> f805861... issue #39: rename structs to no longer assume a specific docker-compose schema version
func ParseHealthcheck(healthcheckYAML *ServiceHealthcheck) (*Healthcheck, bool, error) {
	if healthcheckYAML == nil {
		return nil, false, nil
	}

	if healthcheckYAML.Disable {
		return nil, true, nil
	}

	// Parse Test
	test := healthcheckYAML.GetTest()
	if len(test) == 0 {
		return nil, false, fmt.Errorf("field \"test\" of Healthcheck must not be an empty array")
	}
	healthcheck := &Healthcheck{}
	switch test[0] {
	case HealthcheckCommandNone:
	case HealthcheckCommandCmd:
	case HealthcheckCommandShell:
		healthcheck.IsShell = true
	default:
		return nil, false, fmt.Errorf("field \"test\" of Healthcheck must have a first element that is one of \"NONE\", \"CMD\" and \"%s\"",
			HealthcheckCommandShell)
	}
	if test[0] == HealthcheckCommandNone {
		return nil, true, nil
	}
	if len(test) == 1 {
		return nil, false, fmt.Errorf("field \"test\" of Healthcheck must have size at least 2 if its first element is not \"NONE\"")
	}
	healthcheck.Test = test[1:]

	// We do not set StartPeriod because it is unsupported in docker-compose 2.1 and
	// should therefore be treated as 0.

	// time.ParseDuration supports a superset of durations compared to docker-compose:
	// https://golang.org/pkg/time/#Duration
	// https://docs.docker.com/compose/compose-file/compose-file-v2/#specifying-durations
	if healthcheckYAML.Interval != nil {
		interval, err := time.ParseDuration(*healthcheckYAML.Interval)
		if err != nil {
			return nil, false, err
		}
		if interval <= 0 {
			return nil, false, fmt.Errorf("field \"interval\" of Healthcheck must not be negative")
		}
		healthcheck.Interval = interval
	} else {
		healthcheck.Interval = HealthcheckDefaultInterval
	}

	if healthcheckYAML.Timeout != nil {
		timeout, err := time.ParseDuration(*healthcheckYAML.Timeout)
		if err != nil {
			return nil, false, err
		}
		if timeout <= 0 {
			return nil, false, fmt.Errorf("field \"timeout\" of Healthcheck must not be negative")
		}
		healthcheck.Timeout = timeout
	} else {
		healthcheck.Timeout = HealthcheckDefaultTimeout
	}

	if healthcheckYAML.Retries != nil {
		healthcheck.Retries = *healthcheckYAML.Retries
	} else {
		healthcheck.Retries = HealthcheckDefaultRetries
	}

	return healthcheck, false, nil
}
