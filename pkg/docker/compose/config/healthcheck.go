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

var errorCommandIsNone = fmt.Errorf("test is NONE")

type Healthcheck struct {
	Interval    time.Duration
	IsShell     bool
	StartPeriod time.Duration
	Retries     uint
	Test        []string
	Timeout     time.Duration
}

func ParseHealthcheck(healthcheckYAML *ServiceHealthcheck) (*Healthcheck, bool, error) {
	if healthcheckYAML == nil {
		return nil, false, nil
	}
	if healthcheckYAML.Disable {
		return nil, true, nil
	}
	healthcheck := &Healthcheck{}
	err := healthcheck.parseTest(healthcheckYAML.GetTest())
	if err != nil {
		if err == errorCommandIsNone {
			return nil, true, nil
		}
		return nil, false, err
	}
	err = healthcheck.parseInterval(healthcheckYAML.Interval)
	if err != nil {
		return nil, false, err
	}
	err = healthcheck.parseTimeout(healthcheckYAML.Timeout)
	if err != nil {
		return nil, false, err
	}
	healthcheck.parseRetries(healthcheckYAML.Retries)
	return healthcheck, false, nil
}

func (healthcheck *Healthcheck) parseTimeout(value *string) error {
	if value != nil {
		var err error
		healthcheck.Timeout, err = time.ParseDuration(*value)
		if err != nil {
			return err
		}
		if healthcheck.Timeout <= 0 {
			return fmt.Errorf("field \"timeout\" of Healthcheck must not be negative")
		}
	} else {
		healthcheck.Timeout = HealthcheckDefaultTimeout
	}
	return nil
}

func (healthcheck *Healthcheck) parseTest(test []string) error {
	// Parse Test
	if len(test) == 0 {
		return fmt.Errorf("field \"test\" of Healthcheck must not be an empty array")
	}
	switch test[0] {
	case HealthcheckCommandNone:
		return errorCommandIsNone
	case HealthcheckCommandCmd:
	case HealthcheckCommandShell:
		healthcheck.IsShell = true
	default:
		return fmt.Errorf("field \"test\" of Healthcheck must have a first element that is one of \"NONE\", \"CMD\" and \"%s\"",
			HealthcheckCommandShell)
	}
	if len(test) == 1 {
		return fmt.Errorf("field \"test\" of Healthcheck must have size at least 2 if its first element is not \"NONE\"")
	}
	healthcheck.Test = test[1:]
	return nil
}

func (healthcheck *Healthcheck) parseInterval(value *string) error {
	// We do not set StartPeriod because it is unsupported in docker-compose 2.1 and
	// should therefore be treated as 0.
	// time.ParseDuration supports a superset of durations compared to docker-compose:
	// https://golang.org/pkg/time/#Duration
	// https://docs.docker.com/compose/compose-file/compose-file-v2/#specifying-durations
	if value != nil {
		interval, err := time.ParseDuration(*value)
		if err != nil {
			return err
		}
		if interval <= 0 {
			return fmt.Errorf("field \"interval\" of Healthcheck must not be negative")
		}
		healthcheck.Interval = interval
	} else {
		healthcheck.Interval = HealthcheckDefaultInterval
	}
	return nil
}

func (healthcheck *Healthcheck) parseRetries(value *uint) {
	if value != nil {
		healthcheck.Retries = *value
	} else {
		healthcheck.Retries = HealthcheckDefaultRetries
	}

}
