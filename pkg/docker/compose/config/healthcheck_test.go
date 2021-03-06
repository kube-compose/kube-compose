package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

func TestParseRetries_Normal(t *testing.T) {
	value := new(uint)
	*value = 10
	h := &Healthcheck{}
	h.parseRetries(value)
	if h.Retries != 10 {
		t.Fail()
	}
}

func TestParseRetries_Default(t *testing.T) {
	h := &Healthcheck{}
	h.parseRetries(nil)
	if h.Retries != HealthcheckDefaultRetries {
		t.Fail()
	}
}

func TestParseInterval_Normal(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseInterval(util.NewString("2m1s"))
	if err != nil {
		t.Error(err)
	}
	if h.Interval != 2*time.Minute+time.Second {
		t.Fail()
	}
}
func TestParseInterval_InvalidDuration(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseInterval(util.NewString("asdf1"))
	if err == nil {
		t.Fail()
	}
}
func TestParseInterval_NegativeDuration(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseInterval(util.NewString("-2m"))
	if err == nil {
		t.Fail()
	}
}

func TestParseInterval_Default(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseInterval(nil)
	if err != nil {
		t.Error(err)
	}
	if h.Interval != HealthcheckDefaultInterval {
		t.Fail()
	}
}

func TestParseTimeout_Normal(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTimeout(util.NewString("1m1s"))
	if err != nil {
		t.Error(err)
	}
	if h.Timeout != time.Minute+time.Second {
		t.Fail()
	}
}
func TestParseTimeout_InvalidDuration(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTimeout(util.NewString("asdf2"))
	if err == nil {
		t.Fail()
	}
}
func TestParseTimeout_NegativeDuration(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTimeout(util.NewString("-1m"))
	if err == nil {
		t.Fail()
	}
}

func TestParseTimeout_Default(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTimeout(nil)
	if err != nil {
		t.Error(err)
	}
	if h.Timeout != HealthcheckDefaultTimeout {
		t.Fail()
	}
}

func TestParseTest_EmptySlice(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTest([]string{})
	if err == nil {
		t.Fail()
	}
}

func TestParseTest_None(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTest([]string{HealthcheckCommandNone})
	if err != errorCommandIsNone {
		t.Error(err)
	}
}

func TestParseTest_ShellSuccess(t *testing.T) {
	h := &Healthcheck{}
	input := []string{HealthcheckCommandShell, "echo 'Hello World!'"}
	err := h.parseTest(input)
	if err != nil {
		t.Error(err)
	}
	if !h.IsShell {
		t.Fail()
	}
	if !reflect.DeepEqual(h.Test, input[1:]) {
		t.Error(h.Test)
	}
}

func TestParseTest_CmdSuccess(t *testing.T) {
	h := &Healthcheck{}
	input := []string{HealthcheckCommandCmd, "/bin/zsh", "-c", "echo 'Hello World!'"}
	err := h.parseTest(input)
	if err != nil {
		t.Error(err)
	}
	if h.IsShell {
		t.Fail()
	}
	if !reflect.DeepEqual(h.Test, input[1:]) {
		t.Error(h.Test)
	}
}

func TestParseTest_CmdAndSliceTooSmall(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTest([]string{HealthcheckCommandCmd})
	if err == nil {
		t.Fail()
	}
}

func TestParseTest_InvalidFirstElement(t *testing.T) {
	h := &Healthcheck{}
	err := h.parseTest([]string{"asdf3"})
	if err == nil {
		t.Fail()
	}
}

func TestParseHealthcheck_Disabled(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Disable: util.NewBool(true),
	}
	_, isDisabled, err := ParseHealthcheck(healthcheckYAML)
	if err != nil {
		t.Error(err)
	}
	if !isDisabled {
		t.Fail()
	}
}
func TestParseHealthcheck_Nil(t *testing.T) {
	healthcheck, isDisabled, err := ParseHealthcheck(nil)
	if err != nil {
		t.Error(err)
	}
	if isDisabled || healthcheck != nil {
		t.Fail()
	}
}
func TestParseHealthcheck_TestNone(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Test: HealthcheckTest{
			Values: []string{HealthcheckCommandNone},
		},
	}
	_, isDisabled, err := ParseHealthcheck(healthcheckYAML)
	if err != nil {
		t.Error(err)
	}
	if !isDisabled {
		t.Fail()
	}
}

func TestParseHealthcheck_TestInvalid(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Test: HealthcheckTest{
			Values: []string{"asdf4"},
		},
	}
	_, _, err := ParseHealthcheck(healthcheckYAML)
	if err == nil {
		t.Fail()
	}
}

func TestParseHealthcheck_IntervalInvalid(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Test: HealthcheckTest{
			Values: []string{HealthcheckCommandShell, "echo 'Hello World 2!'"},
		},
		Interval: util.NewString("asdf5"),
	}
	_, _, err := ParseHealthcheck(healthcheckYAML)
	if err == nil {
		t.Fail()
	}
}
func TestParseHealthcheck_InvalidTimeout(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Test: HealthcheckTest{
			Values: []string{HealthcheckCommandShell, "echo 'Hello World 3!'"},
		},
		Timeout: util.NewString("asdf6"),
	}
	_, _, err := ParseHealthcheck(healthcheckYAML)
	if err == nil {
		t.Fail()
	}
}

func TestParseHealthcheck_Success(t *testing.T) {
	healthcheckYAML := &healthcheckInternal{
		Test: HealthcheckTest{
			Values: []string{HealthcheckCommandShell, "echo 'Hello World 4!'"},
		},
	}
	healthcheck, isDisabled, err := ParseHealthcheck(healthcheckYAML)
	if err != nil {
		t.Error(err)
	}
	if isDisabled {
		t.Fail()
	}
	if !reflect.DeepEqual(*healthcheck, Healthcheck{
		Interval: HealthcheckDefaultInterval,
		IsShell:  true,
		Retries:  HealthcheckDefaultRetries,
		Test:     []string{"echo 'Hello World 4!'"},
		Timeout:  HealthcheckDefaultTimeout,
	}) {
		t.Errorf("%+v\n", *healthcheck)
	}
}
