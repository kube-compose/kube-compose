package up

import (
	"math"
	"reflect"
	"testing"
	"time"

	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
	v1 "k8s.io/api/core/v1"
)

func Test_CreateReadinessProbeFromDockerHealthcheck_NilSuccess(t *testing.T) {
	if createReadinessProbeFromDockerHealthcheck(nil) != nil {
		t.Fail()
	}
}

func Test_CreateReadinessProbeFromDockerHealthcheck_SuccessNonNil1(t *testing.T) {
	probe := createReadinessProbeFromDockerHealthcheck(&dockerComposeConfig.Healthcheck{
		Retries: math.MaxUint32,
	})
	if probe.FailureThreshold != math.MaxInt32 {
		t.Fail()
	}
}

func Test_CreateReadinessProbeFromDockerHealthcheck_Success(t *testing.T) {
	arg := "echo 'Hello World!'"
	probe := createReadinessProbeFromDockerHealthcheck(&dockerComposeConfig.Healthcheck{
		IsShell: true,
		Retries: 3,
		Test:    []string{arg},
	})
	if !reflect.DeepEqual(probe, &v1.Probe{
		FailureThreshold: 3,
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{"/bin/sh", "-c", arg},
			},
		},
	}) {
		t.Fail()
	}
}

func Test_InspectImageRawParseHealthcheck_ErrorInvalidJson(t *testing.T) {
	inspectRaw := []byte("{")
	_, err := inspectImageRawParseHealthcheck(inspectRaw)
	if err == nil {
		t.Fail()
	}
}

func Test_InspectImageRawParseHealthcheck_SuccessCommandNone(t *testing.T) {
	inspectRaw := []byte(`{"Config":{"Healthcheck":{"Test":["` + dockerComposeConfig.HealthcheckCommandNone + `"]}}}`)
	healthcheck, err := inspectImageRawParseHealthcheck(inspectRaw)
	if err != nil {
		t.Error(err)
	} else if healthcheck != nil {
		t.Fail()
	}
}
func Test_InspectImageRawParseHealthcheck_Success(t *testing.T) {
	inspectRaw := []byte(`{"Config":{"Healthcheck":{"Test":["` +
		dockerComposeConfig.HealthcheckCommandShell +
		`","arg"],"Timeout":1,"Interval":1,"Retries":1}}}`)
	healthcheck, err := inspectImageRawParseHealthcheck(inspectRaw)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(healthcheck, &dockerComposeConfig.Healthcheck{
		Test:     []string{"arg"},
		Interval: time.Duration(1),
		IsShell:  true,
		Timeout:  time.Duration(1),
		Retries:  1,
	}) {
		t.Logf("%+v\n", healthcheck)
		t.Fail()
	}
}
