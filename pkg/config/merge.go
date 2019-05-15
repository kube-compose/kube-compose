package config

// extendedService is merged into service
func merge(service, extendedService *Service) {
	// rules based on https://docs.docker.com/compose/extends/#adding-and-overriding-configuration
	mergeStringMap(service.Environment, extendedService.Environment)
	service.Ports = mergePortBindings(service.Ports, extendedService.Ports)
	// TODO https://github.com/jbrekelmans/kube-compose/issues/48
}

func mergeStringMap(intoStringMap, fromStringMap map[string]string) {
	for k, v := range fromStringMap {
		if _, ok := intoStringMap[k]; !ok {
			intoStringMap[k] = v
		}
	}
}

func mergePortBindings(intoPorts []PortBinding, fromPorts []PortBinding) []PortBinding {
	for _, v := range fromPorts {
		intoPorts = appendPortBindingIfUnique(intoPorts, v)
	}
	return intoPorts
}

func appendPortBindingIfUnique(slice []PortBinding, port PortBinding) []PortBinding {
	for _, element := range slice {
		if element == port {
			return slice
		}
	}
	return append(slice, port)
}
