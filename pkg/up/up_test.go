package up

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	dockerComposeConfig "github.com/jbrekelmans/kube-compose/pkg/docker/compose/config"
)

const (
	TestRestartPolicyAlways    = "Always"
	TestRestartPolicyOnFailure = "OnFailure"
	TestRestartPolicyNever     = "Never"
)

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.AddService("a", &dockerComposeConfig.Service{
		Restart: "no",
	})
	cfg.AddService("b", &dockerComposeConfig.Service{
		Restart: "always",
	})
	cfg.AddService("c", &dockerComposeConfig.Service{
		Restart: "on-failure",
	})
	cfg.AddService("d", &dockerComposeConfig.Service{})
	return cfg
}

func newTestApp(serviceName string) *app {
	cfg := newTestConfig()
	app := &app{
		composeService: cfg.FindServiceByName(serviceName),
	}
	return app
}
func TestRestartPolicyforService_Never(t *testing.T) {
	app := newTestApp("a")
	restartPolicy := getRestartPolicyforService(app)
	if restartPolicy != TestRestartPolicyNever {
		t.Fail()
	}
}

func TestRestartPolicyforService_Always(t *testing.T) {
	app := newTestApp("b")
	restartPolicy := getRestartPolicyforService(app)
	if restartPolicy != TestRestartPolicyAlways {
		t.Fail()
	}
}
func TestRestartPolicyforService_Onfailure(t *testing.T) {
	app := newTestApp("c")
	restartPolicy := getRestartPolicyforService(app)
	if restartPolicy != TestRestartPolicyOnFailure {
		t.Fail()
	}
}
func TestRestartPolicyforService_Default(t *testing.T) {
	app := newTestApp("d")
	restartPolicy := getRestartPolicyforService(app)
	if restartPolicy != TestRestartPolicyNever {
		t.Fail()
	}
}

func TestAppName(t *testing.T) {
	app := newTestApp("a")
	if app.name() != "a" {
		t.Fail()
	}
}

func TestAppHasService_False(t *testing.T) {
	app := newTestApp("a")
	if app.hasService() {
		t.Fail()
	}
}
func TestAppHasService_True(t *testing.T) {
	app := newTestApp("a")
	app.composeService.Ports = []config.Port{
		config.Port{
			Port:     1234,
			Protocol: "tcp",
		},
	}
	if !app.hasService() {
		t.Fail()
	}
}
