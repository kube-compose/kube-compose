package config

// extendedService is merged into service
func merge(service *Service, extendedService *Service) {
	// rules based on https://docs.docker.com/compose/extends/#adding-and-overriding-configuration
	mergeMap(service.Environment, extendedService.Environment)
	if len(extendedService.Image) > 0 {
		service.Image = extendedService.Image
	}
}

func mergeMap(env map[string]string, extendedEnv map[string]string) {
	for k, v := range extendedEnv {
		env[k] = v
	}
}
