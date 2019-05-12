package k8smeta

import (
	"fmt"
	"strings"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnnotationName is the name of an annotation added by kube compose to resources, so that resources can be mapped back to their docker
// compose service.
const AnnotationName = "kube-compose/service"

// ErrorResourcesModifiedExternally returns an error indicating that resources managed by kube-compose have been modified externally.
func ErrorResourcesModifiedExternally() error {
	return fmt.Errorf("one or more resources appear to have been modified by an external process, aborting")
}

func validateComposeService(cfg *config.Config, composeService *config.Service) {
	if composeService != cfg.CanonicalComposeFile.Services[composeService.ServiceName] {
		panic(fmt.Errorf("invalid service"))
	}
}

// InitCommonLabels adds the labels for the specified docker compose service to the string map.
func InitCommonLabels(cfg *config.Config, composeService *config.Service, labels map[string]string) map[string]string {
	validateComposeService(cfg, composeService)
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app"] = EscapeName(composeService.ServiceName)
	labels[cfg.EnvironmentLabel] = cfg.EnvironmentID
	return labels
}

// InitObjectMeta sets the name, labels and annotations of a resource for the specified docker compose service.
func InitObjectMeta(cfg *config.Config, objectMeta *metav1.ObjectMeta, composeService *config.Service) {
	validateComposeService(cfg, composeService)
	objectMeta.Name = EscapeName(composeService.ServiceName) + "-" + cfg.EnvironmentID
	objectMeta.Labels = InitCommonLabels(cfg, composeService, objectMeta.Labels)
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = map[string]string{}
	}
	objectMeta.Annotations[AnnotationName] = composeService.ServiceName
}

// FindFromObjectMeta finds a docker compose service from resource metadata.
func FindFromObjectMeta(cfg *config.Config, objectMeta *metav1.ObjectMeta) (*config.Service, error) {
	if objectMeta.Annotations != nil {
		if composeServiceName, ok := objectMeta.Annotations[AnnotationName]; ok {
			composeService := cfg.CanonicalComposeFile.Services[composeServiceName]
			if composeService != nil {
				return composeService, nil
			}
		}
	}
	i := strings.IndexByte(objectMeta.Name, '-')
	if i >= 0 && objectMeta.Name[i+1:] == cfg.EnvironmentID {
		prefix := objectMeta.Name[:i]
		for composeServiceName := range cfg.CanonicalComposeFile.Services {
			if EscapeName(composeServiceName) == prefix {
				return nil, ErrorResourcesModifiedExternally()
			}
		}
	}
	return nil, nil
}
