package config


type healthcheckTest struct {
	Values []string
}

func (t *healthcheckTest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.Values)
	if err != nil {
		var str string
		err = unmarshal(&str)
		if err != nil {
			return err
		}
		t.Values = []string{
			healthcheckCommandShell,
			str,
		}
	}
	return nil
}

type healthcheckCompose2_1 struct{
	Disable bool `yaml:"disable"`
	Interval string `yaml:"interval"`
	Retries uint `yaml:"retries"`
	// start_period is only available in docker-compose 2.3 or higher
	Test healthcheckTest `yaml:"test"`
	Timeout string `yaml:"timeout"`
}

func (h *healthcheckCompose2_1) GetTest() []string {
	return h.Test.Values
}

type serviceYAML2_1 struct{
	Build struct{
		Context string `yaml:"context"`
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"build"`
	Environment map[string]string `yaml:"environment"`
	Healthcheck *healthcheckCompose2_1 `yaml:"healthcheck"`
	Image string `yaml:"image"`
	Ports []string `yaml:"ports"`
	Volumes []string `yaml:"volumes"`
	WorkingDir string `yaml:"working_dir"`
}


type composeYAML2_1 struct {
	Services map[string]serviceYAML2_1 `yaml:"services"`
	Volumes map[string]interface{} `yaml:"volumes"`
}