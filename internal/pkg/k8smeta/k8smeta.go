package k8smeta

import (
	"fmt"

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

// InitCommonLabels adds the labels for the specified docker compose service to the string map.
func InitCommonLabels(cfg *config.Config, composeService *config.Service, labels map[string]string) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app"] = composeService.NameEscaped
	labels[cfg.EnvironmentLabel] = cfg.EnvironmentID
	return labels
}

// InitObjectMeta sets the name, labels and annotations of a resource for the specified docker compose service.
func InitObjectMeta(cfg *config.Config, objectMeta *metav1.ObjectMeta, composeService *config.Service) {
	objectMeta.Name = composeService.NameEscaped + "-" + cfg.EnvironmentID
	objectMeta.Labels = InitCommonLabels(cfg, composeService, objectMeta.Labels)
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = map[string]string{}
	}
	objectMeta.Annotations[AnnotationName] = composeService.Name
}

// FindFromObjectMeta finds a docker compose service from resource metadata.
func FindFromObjectMeta(cfg *config.Config, objectMeta *metav1.ObjectMeta) (*config.Service, error) {
	if composeServiceName, ok := objectMeta.Annotations[AnnotationName]; ok {
		composeService := cfg.FindServiceByName(composeServiceName)
		if composeService != nil {
			return composeService, nil
		}
	}
	return nil, nil
}
