package config

import (
	"testing"
)

func TestConfigLoaderParseEnvironment_Success(t *testing.T) {

}

/*

func (c *configLoader) parseEnvironment(env []environmentNameValuePair) (map[string]string, error) {
	envParsed := make(map[string]string, len(env))
	for _, pair := range env {
		var value string
		if pair.Name == "" {
			return nil, fmt.Errorf("invalid environment variable: %s", pair.Name)
		}
		switch {
		case pair.Value == nil:
			var ok bool
			if value, ok = c.environmentGetter(pair.Name); !ok {
				continue
			}
		case pair.Value.StringValue != nil:
			value = *pair.Value.StringValue
		case pair.Value.Int64Value != nil:
			value = strconv.FormatInt(*pair.Value.Int64Value, 10)
		case pair.Value.FloatValue != nil:
			value = strconv.FormatFloat(*pair.Value.FloatValue, 'g', -1, 64)
		default:
			// Environment variables with null values in the YAML are ignored.
			// See test/docker-compose.null-env.yml.
			continue
		}
		envParsed[pair.Name] = value
	}
	return envParsed, nil
}
*/
