package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	version "github.com/hashicorp/go-version"
	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/pkg/errors"
	"github.com/uber-go/mapdecode"
	yaml "gopkg.in/yaml.v2"
)

var (
	// FS is the file system used by the config package. This is useful for unit testing.
	FS   = fsPackage.OSFileSystem()
	v1   = version.Must(version.NewVersion("1"))
	v2_1 = version.Must(version.NewVersion("2.1"))
	v3_1 = version.Must(version.NewVersion("3.1"))
	v3_3 = version.Must(version.NewVersion("3.3"))
)

// TODO https://github.com/kube-compose/kube-compose/issues/11 ensure that the YAML decoder actually produces this
// type for any YAML where the root is a mapping in the absence of type information.
type genericMap map[interface{}]interface{}

// This type is used to represent extension fields, see https://docs.docker.com/compose/compose-file/#extension-fields.
type XProperties map[string]interface{}

// CanonicalDockerComposeConfig is a canonical representation of docker compose configuration.
// It represents one ore more docker compose files that have been merged together using logic close to docker compose.
// Similarly, extends will have been processed as well (see https://docs.docker.com/compose/compose-file/compose-file-v2/#extends).
type CanonicalDockerComposeConfig struct {
	Services    map[string]*Service
	XProperties XProperties
}

// Service is the final representation of a docker-compose service, after all docker compose files have been merged. Service
// is a smaller piece of CanonicalDockerComposeConfig.
type Service struct {
	Command    []string
	DependsOn  map[*Service]ServiceHealthiness
	Entrypoint []string

	// docker-compose distinguishes between an empty Entrypoint and an absent Entrypoint.
	// TODO https://github.com/kube-compose/kube-compose/issues/157 remove this field
	EntrypointPresent   bool
	Environment         map[string]string
	Healthcheck         *Healthcheck
	HealthcheckDisabled bool
	Image               string
	Ports               []PortBinding
	User                *string
	Volumes             []ServiceVolume
	WorkingDir          string
	Restart             string
}

// composeFileParsedService is a helper struct that is a smaller piece of composeFileParsed.
type composeFileParsedService struct {
	service   *Service
	dependsOn map[string]ServiceHealthiness
	extends   *extends

	// Helper data used to detect cycles during process of extends and depends_on.
	recStack bool
	visited  bool
}

// A helper for defer
func (c *composeFileParsedService) clearRecStack() {
	c.recStack = false
}

// composeFileParsed is an intermediate representation of a docker compose file used during loading
// of the docker compose configuration.
type composeFileParsed struct {
	services map[string]*composeFileParsedService
	version  *version.Version
	// Extension fields at the root of the compose file represented by this struct.
	xProperties XProperties
	// The resolved file that contains the docker compose file represented by this struct.
	// Used to resolve files relative to this configuration file, and used when determining the order
	// in which to merge slices.
	resolvedFile string
}

// loadResolvedFileCacheItem is used for cache entries.
type loadResolvedFileCacheItem struct {
	parsed *composeFileParsed
	err    error
}

type configLoader struct {
	environmentGetter ValueGetter
	// A cache required to detect cycles when processing extends. Additionally, each file is only
	// processed once so that loading of configuration is faster.
	loadResolvedFileCache map[string]*loadResolvedFileCacheItem
}

func loadFileError(file string, err error) error {
	return errors.Wrap(err, fmt.Sprintf("error loading file %#v", file))
}

// loadFile loads the specified file. If the file has already been loaded then a cache lookup is performed.
// If file is relative then it is interpreted relative to the current working directory.
func (c *configLoader) loadFile(file string) (*composeFileParsed, error) {
	resolvedFile, err := FS.EvalSymlinks(file)
	if err != nil {
		return nil, loadFileError(file, err)
	}
	return c.loadResolvedFile(resolvedFile)
}

// getVersion is a utility used to retrieve the version from a docker compose file after it has been mapdecode'd.
func getVersion(dataMap genericMap) (*version.Version, error) {
	var v *version.Version
	vRaw, hasVersion := dataMap["version"]
	if !hasVersion {
		v = v1
	} else if vString, ok := vRaw.(string); ok {
		var err error
		v, err = version.NewVersion(vString)
		if err != nil {
			return nil, fmt.Errorf("could not parse version: %#v", vString)
		}
	} else {
		return nil, fmt.Errorf("version must be a string")
	}
	return v, nil
}

// loadResolvedFile is a wrapper around loadResolvedFileCore that loads and populates a cache.
func (c *configLoader) loadResolvedFile(resolvedFile string) (*composeFileParsed, error) {
	cacheItem := c.loadResolvedFileCache[resolvedFile]
	if cacheItem == nil {
		// Add an item to the cache before loadResolvedFileCore so that a recursive call within loadResolvedFileCore
		// can detect cycles.
		cacheItem = &loadResolvedFileCacheItem{
			parsed: &composeFileParsed{},
		}
		c.loadResolvedFileCache[resolvedFile] = cacheItem
		cacheItem.err = c.loadResolvedFileCore(resolvedFile, cacheItem.parsed)

	}
	return cacheItem.parsed, cacheItem.err
}

// loadYamlFileAsGenericMap is a helper used to YAML decode a file into a map[interface{}]interface{}.
func loadYamlFileAsGenericMap(file string) (genericMap, error) {
	reader, err := FS.Open(file)
	if err != nil {
		return nil, err
	}
	defer util.CloseAndLogError(reader)
	decoder := yaml.NewDecoder(reader)
	var dataMap genericMap
	err = decoder.Decode(&dataMap)
	return dataMap, err
}

// loadResolvedFileCore loads a docker compose file, and does any validation/canonicalization that does not require
// knowledge of other services. In other words, extends and depends_on are not processed by loadResolvedFileCore.
func (c *configLoader) loadResolvedFileCore(resolvedFile string, cfParsed *composeFileParsed) error {
	cfParsed.resolvedFile = resolvedFile

	// Load YAML file as map[interface{}]interface{}. This type is used so that we can subsequently
	// interpolate environment variables and extract x- properties.
	dataMap, err := loadYamlFileAsGenericMap(resolvedFile)
	if err != nil {
		return err
	}

	// extract docker compose file version
	cfParsed.version, err = getVersion(dataMap)
	if err != nil {
		return err
	}

	// Substitute variables with environment variables.
	err = InterpolateConfig(dataMap, c.environmentGetter, cfParsed.version)
	if err != nil {
		return err
	}

	if !cfParsed.version.Equal(v1) {
		// extract x- properties
		cfParsed.xProperties = getXProperties(dataMap)
	} else {
		dataMap = map[interface{}]interface{}{
			"services": dataMap,
		}
	}

	// mapdecode based on docker compose file schema
	var cf composeFile
	err = mapdecode.Decode(&cf, dataMap, mapdecode.IgnoreUnused(true))
	if err != nil {
		return err
	}

	// validation after parsing
	return c.parseComposeFile(&cf, cfParsed)
}

// loadStandardFile loads the docker compose file at a standard location.
func (c *configLoader) loadStandardFile() (*composeFileParsed, error) {
	file := "docker-compose.yml"
	resolvedFile, err := FS.EvalSymlinks(file)
	if os.IsNotExist(err) {
		file = "docker-compose.yaml"
		resolvedFile, err = FS.EvalSymlinks(file)
	}
	if err == nil {
		return c.loadResolvedFile(resolvedFile)
	}
	return nil, loadFileError(file, err)
}

// processExtends process the extends field of a docker compose service. That is: given a docker compose service X named name in the docker
// compose file cfParsed.resolvedFile, if X extends another service Y then processExtends copies inherited configuration Y into the
// representation of X (cfServiceParsed).
func (c *configLoader) processExtends(name string, cfServiceParsed *composeFileParsedService, cfParsed *composeFileParsed) error {
	if cfServiceParsed.visited {
		if cfServiceParsed.recStack {
			return fmt.Errorf("cannot extend service %s of file %#v because this would cause an infinite loop. Please ensure your docker "+
				"compose services do not have a cyclical extends relationship", name, cfParsed.resolvedFile)
		}
		return nil
	}
	cfServiceParsed.visited = true
	if cfServiceParsed.extends == nil {
		return nil
	}
	cfServiceParsed.recStack = true
	defer cfServiceParsed.clearRecStack()
	cfExtendedServiceParsed, err := c.resolveExtends(name, cfServiceParsed, cfParsed)
	if err != nil {
		return err
	}
	merge(cfServiceParsed, cfExtendedServiceParsed)
	return nil
}

// resolveExtends ensures the configuration of an extended docker compose service has been loaded.
// This may involve loading another file, and will recursively process any extends, erroring if a cycle is detected.
// More formatlly: given a docker compose service X named name
// in the docker compose file cfParsed.resolvedFile, if X extends another service Y then resolveExtends ensures that:
// 1. the representation of Y has been loaded; -and
// 2. processExtends has been called on Y.
func (c *configLoader) resolveExtends(
	name string,
	cfServiceParsed *composeFileParsedService,
	cfParsed *composeFileParsed) (*composeFileParsedService, error) {
	var cfExtendedServiceParsed *composeFileParsedService
	var cfParsedExtends *composeFileParsed
	if cfServiceParsed.extends.File != nil {
		extendsFile := *cfServiceParsed.extends.File
		if !filepath.IsAbs(extendsFile) {
			extendsFile = filepath.Join(filepath.Dir(cfParsed.resolvedFile), extendsFile)
		}
		var err error
		cfParsedExtends, err = c.loadFile(extendsFile)
		if err != nil {
			return nil, err
		}
		cfExtendedServiceParsed = cfParsedExtends.services[cfServiceParsed.extends.Service]
		if cfExtendedServiceParsed == nil {
			return nil, fmt.Errorf(
				"a service named %s extends non-existent service %s of file %#v",
				name,
				cfServiceParsed.extends.Service,
				cfParsedExtends.resolvedFile,
			)
		}
	} else {
		cfExtendedServiceParsed = cfParsed.services[cfServiceParsed.extends.Service]
		if cfExtendedServiceParsed == nil {
			return nil, fmt.Errorf("a service named %s extends non-existent service %s",
				name,
				cfServiceParsed.extends.Service,
			)
		}
		cfParsedExtends = cfParsed
	}
	if cfExtendedServiceParsed.dependsOn != nil {
		return nil, fmt.Errorf("cannot extend service %s: services with 'depends_on' cannot be extended",
			cfServiceParsed.extends.Service,
		)
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/122 perform full validation of extended service
	err := c.processExtends(cfServiceParsed.extends.Service, cfExtendedServiceParsed, cfParsedExtends)
	if err != nil {
		return nil, err
	}
	return cfExtendedServiceParsed, nil
}

// New loads docker compose configuration from a slice of files.
// If files is an empty slice then the standard docker compose file locations (relative to the current working directory are considered).
func New(files []string) (*CanonicalDockerComposeConfig, error) {
	c := &configLoader{
		environmentGetter:     os.LookupEnv,
		loadResolvedFileCache: map[string]*loadResolvedFileCacheItem{},
	}
	var resolvedFiles []string
	if len(files) > 0 {
		for _, file := range files {
			cfParsed, err := c.loadFile(file)
			if err != nil {
				return nil, err
			}
			resolvedFiles = append(resolvedFiles, cfParsed.resolvedFile)
		}
	} else {
		cfParsed, err := c.loadStandardFile()
		if err != nil {
			return nil, err
		}
		resolvedFiles = append(resolvedFiles, cfParsed.resolvedFile)
	}

	if len(resolvedFiles) > 1 {
		// TODO https://github.com/kube-compose/kube-compose/issues/121 merge files together
		// This should be a matter of calling merge repeatedly.
		return nil, fmt.Errorf("sorry, merging multiple docker compose files is not supported")
	}
	cfParsed := c.loadResolvedFileCache[resolvedFiles[0]].parsed
	for name, cfServiceParsed := range cfParsed.services {
		err := c.processExtends(name, cfServiceParsed, cfParsed)
		if err != nil {
			return nil, err
		}
	}
	err := resolveDependsOn(cfParsed)
	if err != nil {
		return nil, err
	}

	// TODO https://github.com/kube-compose/kube-compose/issues/165 resolve named volumes
	// TODO https://github.com/kube-compose/kube-compose/issues/166 error on duplicate mount points

	configCanonical := &CanonicalDockerComposeConfig{}
	configCanonical.Services = map[string]*Service{}
	for name, cfServiceParsed := range cfParsed.services {
		configCanonical.Services[name] = cfServiceParsed.service
	}
	configCanonical.XProperties = cfParsed.xProperties
	return configCanonical, nil
}

// getXProperties is a utility that gets all string properties starting with x- from gm, if gm is of type map[interface{}]interface{}.
func getXProperties(gm interface{}) XProperties {
	gmMap, ok := gm.(genericMap)
	if !ok {
		return nil
	}
	var result XProperties
	for key, value := range gmMap {
		keyString, ok := key.(string)
		if ok && strings.HasPrefix(keyString, "x-") {
			if result == nil {
				result = XProperties{}
			}
			result[keyString] = value
		}
	}
	return result
}

func resolveDependsOn(cfParsed *composeFileParsed) error {
	for name1, cfServiceParsed := range cfParsed.services {
		service := cfServiceParsed.service
		service.DependsOn = map[*Service]ServiceHealthiness{}
		for name2, serviceHealthiness := range cfServiceParsed.dependsOn {
			resolvedDependsOn := cfParsed.services[name2]
			if resolvedDependsOn == nil {
				return fmt.Errorf("service %s refers to a non-existing service in its depends_on: %s",
					name1, name2)
			}
			service.DependsOn[resolvedDependsOn.service] = serviceHealthiness
		}
	}
	for name, cfServiceParsed := range cfParsed.services {
		// Reset the visited marker on each service. This is a precondition of ensureNoDependsOnCycle.
		for _, cfServiceParsed := range cfParsed.services {
			cfServiceParsed.visited = false
		}
		// Run the cycle detection algorithm...
		err := ensureNoDependsOnCycle(name, cfServiceParsed, cfParsed)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://www.geeksforgeeks.org/detect-cycle-in-a-graph/
func ensureNoDependsOnCycle(name1 string, cfServiceParsed *composeFileParsedService, cfParsed *composeFileParsed) error {
	cfServiceParsed.visited = true
	cfServiceParsed.recStack = true
	defer cfServiceParsed.clearRecStack()
	for name2 := range cfServiceParsed.dependsOn {
		dependsOn := cfParsed.services[name2]
		if !dependsOn.visited {
			err := ensureNoDependsOnCycle(name2, dependsOn, cfParsed)
			if err != nil {
				return err
			}
		} else if dependsOn.recStack {
			return fmt.Errorf("a service %s depends on a service %s, but this means there is a cycle in the depends_on relationship",
				name1, name2)
		}
	}
	return nil
}

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
func (c *configLoader) parseComposeFile(cf *composeFile, cfParsed *composeFileParsed) error {
	cfParsed.services = make(map[string]*composeFileParsedService, len(cf.Services))
	for name, cfService := range cf.Services {
		composeFileParsedService, err := c.parseComposeFileService(cfParsed.resolvedFile, cfService)
		if err != nil {
			return err
		}
		cfParsed.services[name] = composeFileParsedService
	}
	return nil
}

func (c *configLoader) parseComposeFileService(resolvedFile string, cfService *composeFileService) (*composeFileParsedService, error) {
	service := &Service{
		Command:    cfService.Command.Values,
		Image:      cfService.Image,
		User:       cfService.User,
		Volumes:    cfService.Volumes,
		WorkingDir: cfService.WorkingDir,
		Restart:    cfService.Restart,
	}
	composeFileParsedService := &composeFileParsedService{
		service: service,
		extends: cfService.Extends,
	}
	if cfService.Entrypoint != nil {
		service.Entrypoint = cfService.Entrypoint.Values
		service.EntrypointPresent = true
	}
	if cfService.DependsOn != nil {
		composeFileParsedService.dependsOn = cfService.DependsOn.Values
	}
	ports, err := parsePorts(cfService.Ports)
	if err != nil {
		return nil, err
	}
	service.Ports = ports

	healthcheck, healthcheckDisabled, err := ParseHealthcheck(cfService.Healthcheck)
	if err != nil {
		return nil, err
	}
	service.Healthcheck = healthcheck
	service.HealthcheckDisabled = healthcheckDisabled

	environment, err := c.parseEnvironment(cfService.Environment.Values)
	if err != nil {
		return nil, err
	}
	service.Environment = environment

	// TODO https://github.com/kube-compose/kube-compose/issues/163 only resolve volume paths if volume_driver is not set.
	for i := 0; i < len(service.Volumes); i++ {
		err = resolveHostPath(resolvedFile, &service.Volumes[i])
		if err != nil {
			return nil, err
		}
	}

	return composeFileParsedService, nil
}

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
