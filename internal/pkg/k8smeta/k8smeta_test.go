package k8smeta

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindFromObjectMeta_AnnotationSuccess(t *testing.T) {
	serviceA := &config.Service{
		Name: "a",
	}
	cfg := config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{
				serviceA.Name: serviceA,
			},
		},
	}
	objectMeta := metav1.ObjectMeta{
		Annotations: map[string]string{
			AnnotationName: serviceA.Name,
		},
	}
	composeService, err := FindFromObjectMeta(&cfg, &objectMeta)
	if err != nil {
		t.Fail()
	}
	if composeService != serviceA {
		t.Fail()
	}
}

func TestFindFromObjectMeta_ResourceModifiedExternally(t *testing.T) {
	serviceA := &config.Service{
		Name: "a",
	}
	cfg := config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{
				serviceA.Name: serviceA,
			},
		},
		EnvironmentID: "myenv",
	}
	objectMeta := metav1.ObjectMeta{
		Name: "a-myenv",
	}
	_, err := FindFromObjectMeta(&cfg, &objectMeta)
	if err == nil {
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
	serviceA := &config.Service{
		Name: "a",
	}
	cfg := config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{
				serviceA.Name: serviceA,
			},
		},
	}
	objectMeta := metav1.ObjectMeta{}
	InitObjectMeta(&cfg, &objectMeta, serviceA)
}

func TestInitObjectMeta_InvalidService(t *testing.T) {
	serviceA := &config.Service{
		Name: "a",
	}
	cfg := config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{},
		},
	}
	objectMeta := metav1.ObjectMeta{}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic due to invalid service")
		}
	}()
	InitObjectMeta(&cfg, &objectMeta, serviceA)
}
