package k8smeta

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	dockerComposeConfig "github.com/jbrekelmans/kube-compose/pkg/docker/compose/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.AddService("a", &dockerComposeConfig.Service{})
	return cfg
}

func TestFindFromObjectMeta_AnnotationSuccess(t *testing.T) {
	cfg := newTestConfig()
	serviceA := cfg.FindServiceByName("a")
	objectMeta := metav1.ObjectMeta{
		Annotations: map[string]string{
			AnnotationName: serviceA.Name,
		},
	}
	composeService, err := FindFromObjectMeta(cfg, &objectMeta)
	if err != nil {
		t.Fail()
	}
	if composeService != serviceA {
		t.Fail()
	}
}

func TestFindFromObjectMeta_NotFound(t *testing.T) {
	cfg := config.Config{}
	objectMeta := metav1.ObjectMeta{}
	composeService, err := FindFromObjectMeta(&cfg, &objectMeta)
	if composeService != nil || err != nil {
		t.Fail()
	}
}

func TestInitObjectMeta_Success(t *testing.T) {
	cfg := &config.Config{
		EnvironmentID: "myenv",
	}
	serviceA := cfg.AddService("a", &dockerComposeConfig.Service{})
	objectMeta := metav1.ObjectMeta{}
	InitObjectMeta(cfg, &objectMeta, serviceA)
}

func Test_ErrorResourcesModifiedExternally(t *testing.T) {
	err := ErrorResourcesModifiedExternally()
	if err == nil {
		t.Fail()
	}
}