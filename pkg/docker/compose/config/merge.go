package config

func merge(into, from *composeFileParsedService) {
	// Rules here are based on https://docs.docker.com/compose/extends/#adding-and-overriding-configuration
	mergeStringMaps(into.service.Environment, from.service.Environment)
	into.service.Ports = mergePortBindings(into.service.Ports, from.service.Ports)
	// TODO https://github.com/kube-compose/kube-compose/issues/48 add missing rules here
	// TODO https://github.com/kube-compose/kube-compose/issues/164 support merging of volumes
}

func mergeStringMaps(intoStringMap, fromStringMap map[string]string) {
	for k, v := range fromStringMap {
		if _, ok := intoStringMap[k]; !ok {
			intoStringMap[k] = v
		}
	}
}

func mergePortBindings(intoPorts, fromPorts []PortBinding) []PortBinding {
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
