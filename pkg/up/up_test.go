package up

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/pkg/config"
)

func newTestConfig() *config.Config {
	serviceA := &config.Service{
		Name:    "a",
		Restart: "no",
	}
	serviceB := &config.Service{
		Name:    "b",
		Restart: "always",
	}
	serviceC := &config.Service{
		Name:    "c",
		Restart: "on-failure",
	}
	serviceD := &config.Service{
		Name: "d",
	}

	cfg := &config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{
				serviceA.Name: serviceA,
				serviceB.Name: serviceB,
				serviceC.Name: serviceC,
				serviceD.Name: serviceD,
			},
		},
	}
	return cfg
}

func newTestApp(serviceName string) *app {
	cfg := newTestConfig()
	app := &app{
		composeService: cfg.CanonicalComposeFile.Services[serviceName],
	}
	return app
}
func TestRestartPolicyforService_never(t *testing.T) {
	app := newTestApp("a")
	str := getRestartPolicyforService(app)
	if str != "Never" {
		t.Fail()
	}
}
func TestRestartPolicyforService_always(t *testing.T) {
	app := newTestApp("b")
	str := getRestartPolicyforService(app)
	if str != "Always" {
		t.Fail()
	}
}
func TestRestartPolicyforService_onfailure(t *testing.T) {
	app := newTestApp("c")
	str := getRestartPolicyforService(app)
	if str != "OnFailure" {
		t.Fail()
	}
}
func TestRestartPolicyforService_default(t *testing.T) {
	app := newTestApp("d")
	str := getRestartPolicyforService(app)
	if str != "Never" {
		t.Fail()
	}
}
