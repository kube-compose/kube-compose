package k8smeta

import (
	"testing"

	"github.com/kube-compose/kube-compose/internal/app/config"
	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.AddService(&dockerComposeConfig.Service{
		Name: "a",
	})
	return cfg
}

func TestFindFromObjectMeta_AnnotationSuccess(t *testing.T) {
	cfg := newTestConfig()
	serviceA := cfg.Services["a"]
	objectMeta := metav1.ObjectMeta{
		Annotations: map[string]string{
			AnnotationName: serviceA.Name(),
		},
	}
	composeService := FindFromObjectMeta(cfg, &objectMeta)
	if composeService != serviceA {
		t.Fail()
	}
}

func TestGetK8sName(t *testing.T) {
	service := &config.Service{NameEscaped: "Test"}
	cfg := &config.Config{EnvironmentID: "123"}
	serviceName := GetK8sName(service, cfg)
	if serviceName != "Test-123" {
		t.Fail()
	}
}

func TestFindFromObjectMeta_NotFound(t *testing.T) {
	cfg := config.Config{}
	objectMeta := metav1.ObjectMeta{}
	composeService := FindFromObjectMeta(&cfg, &objectMeta)
	if composeService != nil {
		t.Fail()
	}
}

func TestInitObjectMeta_Success(t *testing.T) {
	cfg := &config.Config{
		EnvironmentID: "myenv",
	}
	serviceA := cfg.AddService(&dockerComposeConfig.Service{
		Name: "a",
	})
	objectMeta := metav1.ObjectMeta{}
	InitObjectMeta(cfg, &objectMeta, serviceA)
}

func Test_ErrorResourcesModifiedExternally(t *testing.T) {
	err := ErrorResourcesModifiedExternally()
	if err == nil {
		t.Fail()
	}
}
