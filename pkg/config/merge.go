package config

// base is
func merge(service *Service, extendedService *Service) {
	mergeEnvironment(service.Environment, extendedService.Environment)
}

func mergeEnvironment(env map[string]string, extendedEnv map[string]string) {

}
