package config

func addPortBinding(ports []PortBinding, port1 PortBinding) []PortBinding {
	for _, port2 := range ports {
		if port1 == port2 {
			return ports
		}
	}
	return append(ports, port1)
}

// Same logic for equal volumes as https://github.com/docker/compose/blob/99e67d0c061fa3d9b9793391f3b7c8bdf8e841fc/compose/config/config.py#L1423
func addVolume(volumes []ServiceVolume, volume1 ServiceVolume) []ServiceVolume {
	for _, volume2 := range volumes {
		if volume2.Short.ContainerPath == volume1.Short.ContainerPath {
			return volumes
		}
	}
	return append(volumes, volume1)
}

func merge(into, from *serviceInternal, mergeExtends bool) {
	// Rules here are based on https://docs.docker.com/compose/extends/#adding-and-overriding-configuration
	into.dependsOn.Values = mergeDependsOnMaps(into.dependsOn.Values, from.dependsOn.Values)
	into.environmentParsed = mergeStringMaps(into.environmentParsed, from.environmentParsed)
	into.healthcheck = mergeHealthchecks(into.healthcheck, from.healthcheck)
	into.portsParsed = mergePortBindings(into.portsParsed, from.portsParsed)
	into.volumes = mergeVolumes(into.volumes, from.volumes)

	if into.entrypoint.Values == nil {
		into.entrypoint.Values = from.entrypoint.Values
	}
	if into.image == nil {
		into.image = from.image
	}
	if into.privileged == nil {
		into.privileged = from.privileged
	}
	if into.restart == nil {
		into.restart = from.restart
	}
	if into.user == nil {
		into.user = from.user
	}
	if mergeExtends && into.extends == nil {
		into.extends = from.extends
	}
}

func mergeDependsOnMaps(into, from map[string]ServiceHealthiness) map[string]ServiceHealthiness {
	if into == nil {
		return from
	}
	for k, v := range from {
		if _, ok := into[k]; !ok {
			into[k] = v
		}
	}
	return into
}

func mergeHealthchecks(into, from *healthcheckInternal) *healthcheckInternal {
	if into == nil {
		return from
	}
	if into.Disable != nil && *into.Disable {
		return into
	}
	if into.Disable == nil {
		into.Disable = from.Disable
	}
	if into.Interval == nil {
		into.Interval = from.Interval
	}
	if into.Retries == nil {
		into.Retries = from.Retries
	}
	// Test.Values is nil if and only if the field is not set. We need to know whether the field is set to correctly merge. See also
	// healthcheckInternal.
	if into.Test.Values == nil {
		into.Test.Values = from.Test.Values
	}
	if into.Timeout == nil {
		into.Timeout = from.Timeout
	}
	return into
}

func mergePortBindings(into, from []PortBinding) []PortBinding {
	if len(into) == 0 {
		return from
	}
	for _, v := range from {
		into = addPortBinding(into, v)
	}
	return into
}

// This merge function is used when merging files together.
// from is never mutated, any any non-frozen/mutable value in from is guaranteed to be copied.
// This is an important correctness property, because from is a pointer to the cache of a docker
// compose file. If from were to be mutated then an extends would inherit from an intermediate result instead of the original docker compose
// file.
func mergeServices(into, from map[string]*serviceInternal) {
	for name, fromService := range from {
		intoService := into[name]
		if intoService == nil {
			intoService = &serviceInternal{
				dependsOn: dependsOn{
					Values: map[string]ServiceHealthiness{},
				},
				environmentParsed: map[string]string{},
				healthcheck:       &healthcheckInternal{},
				portsParsed:       []PortBinding{},
				volumes:           []ServiceVolume{},
			}
			into[name] = intoService
		}
		merge(intoService, fromService, true)
	}
}

func mergeStringMaps(into, from map[string]string) map[string]string {
	if len(into) == 0 {
		return from
	}
	for k, v := range from {
		if _, ok := into[k]; !ok {
			into[k] = v
		}
	}
	return into
}

func mergeVolumes(into, from []ServiceVolume) []ServiceVolume {
	if len(into) == 0 {
		return from
	}
	for _, v := range from {
		into = addVolume(into, v)
	}
	return into
}
