package config

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
type ServiceHealthiness int

const (
	ServiceStarted ServiceHealthiness = 0
	ServiceHealthy ServiceHealthiness = 1
)
