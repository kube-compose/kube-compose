package up

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/internal/app/config"
	dockerComposeConfig "github.com/jbrekelmans/kube-compose/pkg/docker/compose/config"
	"k8s.io/client-go/rest"
)

const (
	TestRestartPolicyAlways    = "Always"
	TestRestartPolicyOnFailure = "OnFailure"
	TestRestartPolicyNever     = "Never"
)

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	serviceA := cfg.AddService("a", &dockerComposeConfig.Service{
		Restart: "no",
	})
	cfg.AddService("b", &dockerComposeConfig.Service{
		Restart: "always",
	})
	serviceC := cfg.AddService("c", &dockerComposeConfig.Service{
		Restart: "on-failure",
	})
	serviceD := cfg.AddService("d", &dockerComposeConfig.Service{})
	serviceA.DockerComposeService.DependsOn = map[*dockerComposeConfig.Service]dockerComposeConfig.ServiceHealthiness{}
	serviceA.DockerComposeService.DependsOn[serviceC.DockerComposeService] = dockerComposeConfig.ServiceHealthy
	serviceA.DockerComposeService.DependsOn[serviceD.DockerComposeService] = dockerComposeConfig.ServiceStarted
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
		{
			Port:     1234,
			Protocol: "tcp",
		},
	}
	if !app.hasService() {
		t.Fail()
	}
}

func TestUpRunnerInitKubernetesClientset(t *testing.T) {
	kubeConfig := &rest.Config{
		Host: "http://localhost:8443/",
	}
	cfg := &config.Config{
		KubeConfig: kubeConfig,
	}
	u := &upRunner{
		cfg: cfg,
	}
	err := u.initKubernetesClientset()
	if err != nil {
		t.Error(err)
	}
}

func TestFormatCreatePodReason(t *testing.T) {
	cfg := newTestConfig()
	u := &upRunner{
		cfg: cfg,
	}
	u.initApps()
	appA := u.apps["a"]
	s := u.formatCreatePodReason(appA)
	if s != "its dependency conditions are met (c: ready, d: running)" {
		t.Error(s)
	}
}
