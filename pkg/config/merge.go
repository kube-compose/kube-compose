package config

// extendedService is merged into service
func merge(service *Service, extendedService *Service) {
	// rules based on https://docs.docker.com/compose/extends/#adding-and-overriding-configuration
	mergeStringMap(service.Environment, extendedService.Environment)
	// TODO https://github.com/jbrekelmans/kube-compose/issues/48
}

func mergeStringMap(intoStringMap map[string]string, fromStringMap map[string]string) {
	for k, v := range fromStringMap {
		if _, ok := intoStringMap[k]; ok {
			intoStringMap[k] = v
		}
	}
}
