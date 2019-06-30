package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/pkg/errors"
	"github.com/uber-go/mapdecode"
	yaml "gopkg.in/yaml.v2"
)

var (
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
	Services map[string]*Service
	// For each docker compose file that was merged together, the root level x- properties as a generic map.
	// Givens elements e_i and e_j of the slice, with indices i and j, respectively, such that i > j, XProperties e_i have a higher priority
	// than XProperties e_j. Intuitively, elements later in the list take precedence over those earlier in the list.
	// The user of this package can choose to implement merging of XProperties as appropriate.
	XProperties []XProperties
}

// Service is the final representation of a docker-compose service, after all docker compose files have been merged. Service
// is a smaller piece of CanonicalDockerComposeConfig.
type Service struct {
	// When adding a field here, please update merge.go with the logic required to merge these fields.
	Command []string
	// TODO https://github.com/kube-compose/kube-compose/issues/214 consider simplifying to map[string]ServiceHealthiness
	DependsOn           map[string]ServiceHealthiness
	Entrypoint          []string
	Environment         map[string]string
	Healthcheck         *Healthcheck
	HealthcheckDisabled bool
	Image               string
	Name                string
	Ports               []PortBinding
	Privileged          bool
	Restart             string
	User                *string
	Volumes             []ServiceVolume
	WorkingDir          string
}

// serviceInternal is a helper struct that is a smaller piece of dockerComposeFile.
// TODO https://github.com/kube-compose/kube-compose/issues/211 merge with composeFileService struct
type serviceInternal struct {
	// TODO https://github.com/kube-compose/kube-compose/issues/153 interpret string command/entrypoint correctly
	Command   *stringOrStringSlice `mapdecode:"command"`
	DependsOn *dependsOn           `mapdecode:"depends_on"`
	// TODO https://github.com/kube-compose/kube-compose/issues/153 interpret string command/entrypoint correctly
	Entrypoint        *stringOrStringSlice `mapdecode:"entrypoint"`
	Environment       *environment         `mapdecode:"environment"`
	environmentParsed map[string]string
	Extends           *extends `mapdecode:"extends"`
	// The final docker compose service in CanonicalDockerComposeConfig (only set if this is not an intermediate result).
	finalService *Service
	Healthcheck  *healthcheckInternal `mapdecode:"healthcheck"`
	Image        *string              `mapdecode:"image"`
	// Convenient copy of the name so that we do not have to pass names around to preserve context.
	name        string
	Ports       []port `mapdecode:"ports"`
	portsParsed []PortBinding
	Privileged  *bool `mapdecode:"privileged"`
	// Helper data used to detect cycles during process of extends and depends_on.
	recStack bool
	Restart  *string `mapdecode:"restart"`
	User     *string `mapdecode:"user"`
	// Helper data used to detect cycles during process of extends and depends_on.
	visited    bool
	Volumes    []ServiceVolume `mapdecode:"volumes"`
	WorkingDir *string         `mapdecode:"working_dir"`
}

// A helper for defer
func (c *serviceInternal) clearRecStack() {
	c.recStack = false
}

// dockerComposeFile is an intermediate representation of a docker compose file used during loading
// of the docker compose configuration.
// TODO https://github.com/kube-compose/kube-compose/issues/211 merge with composeFile struct
type dockerComposeFile struct {
	Services map[string]*serviceInternal `mapdecode:"services"`
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
	parsed *dockerComposeFile
	err    error
}

type configLoader struct {
	environmentGetter ValueGetter
	// A cache required to detect cycles when processing extends. Additionally, each file is only
	// processed once so that loading of configuration is faster.
	loadResolvedFileCache map[string]*loadResolvedFileCacheItem
}

// loadFile loads the specified file. If the file has already been loaded then a cache lookup is performed.
// If file is relative then it is interpreted relative to the current working directory.
func (c *configLoader) loadFile(file string) (*dockerComposeFile, error) {
	resolvedFile, err := fs.OS.EvalSymlinks(file)
	if err != nil {
		return nil, errors.Wrapf(err, "error when evaluating symlinks %#v", file)
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
func (c *configLoader) loadResolvedFile(resolvedFile string) (*dockerComposeFile, error) {
	cacheItem := c.loadResolvedFileCache[resolvedFile]
	if cacheItem == nil {
		// Add an item to the cache before loadResolvedFileCore so that a recursive call within loadResolvedFileCore
		// can detect cycles.
		cacheItem = &loadResolvedFileCacheItem{
			parsed: &dockerComposeFile{},
		}
		c.loadResolvedFileCache[resolvedFile] = cacheItem
		cacheItem.err = c.loadResolvedFileCore(resolvedFile, cacheItem.parsed)
	}
	return cacheItem.parsed, cacheItem.err
}

// loadYamlFileAsGenericMap is a helper used to YAML decode a file into a map[interface{}]interface{}.
func loadYamlFileAsGenericMap(file string) (genericMap, error) {
	reader, err := fs.OS.Open(file)
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
func (c *configLoader) loadResolvedFileCore(resolvedFile string, dcFile *dockerComposeFile) error {
	dcFile.resolvedFile = resolvedFile

	// Load YAML file as map[interface{}]interface{}. This type is used so that we can subsequently
	// interpolate environment variables and extract x- properties.
	dataMap, err := loadYamlFileAsGenericMap(resolvedFile)
	if err != nil {
		return err
	}

	// extract docker compose file version
	dcFile.version, err = getVersion(dataMap)
	if err != nil {
		return err
	}

	// Substitute variables with environment variables.
	err = InterpolateConfig(dataMap, c.environmentGetter, dcFile.version)
	if err != nil {
		return err
	}

	if !dcFile.version.Equal(v1) {
		// extract x- properties
		dcFile.xProperties = getXProperties(dataMap)
	} else {
		dataMap = map[interface{}]interface{}{
			"services": dataMap,
		}
	}
	// mapdecode based on docker compose file schema
	err = mapdecode.Decode(dcFile, dataMap, mapdecode.IgnoreUnused(true))
	if err != nil {
		return err
	}

	// validation after parsing
	return c.parseDockerComposeFile(dcFile)
}

// loadStandardFiles loads the docker compose file at a standard location.
func (c *configLoader) loadStandardFiles() ([]string, error) {
	var resolvedFileSlice []string
	cwd, err := fs.OS.Getwd()
	if err != nil {
		return nil, err
	}
	dir := ""
	resolvedDir, err := fs.OS.EvalSymlinks(cwd)
	if err != nil {
		return nil, err
	}
	for {
		resolvedFile, err := c.loadStandardFilesTry(dir, resolvedDir, "")
		if err == nil {
			resolvedFileSlice = append(resolvedFileSlice, resolvedFile)
			resolvedFile, err = c.loadStandardFilesTry(dir, resolvedDir, ".override")
			if err == nil {
				resolvedFileSlice = append(resolvedFileSlice, resolvedFile)
			}
			if err == nil || os.IsNotExist(err) {
				return resolvedFileSlice, nil
			}
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		resolvedDirParent := filepath.Dir(resolvedDir)
		if resolvedDirParent == resolvedDir {
			break
		}
		resolvedDir = resolvedDirParent
		dir = ".." + string(filepath.Separator) + dir
	}
	return nil, fmt.Errorf("could not find file docker-compose.yml or docker-compose.yaml in (parents of) the current directory %#v", cwd)
}

func (c *configLoader) loadStandardFilesTry(dir, resolvedDir, override string) (resolvedFile string, err error) {
	for _, suffix := range []string{".yml", ".yaml"} {
		basename := "docker-compose" + override + suffix
		file := dir + basename
		resolvedFile, err = fs.OS.EvalSymlinks(resolvedDir + "/" + basename)
		if err == nil {
			_, err = c.loadResolvedFile(resolvedFile)
			if err != nil {
				return "", errors.Wrapf(err, "error while loading docker compose file %s (%#v)", dir+file, resolvedFile)
			}
			return resolvedFile, nil
		}
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "error when evaluating symlinks %s (%#v)", dir+file, resolvedDir+"/"+basename)
		}
	}
	return "", err
}

// processExtends process the extends field of a docker compose service. That is: given a docker compose service X,
// if X extends another service Y then processExtends copies inherited configuration Y into the representation of X.
func (c *configLoader) processExtends(
	s *serviceInternal,
	dcFile *dockerComposeFile) error {
	if s.visited {
		if s.recStack {
			if dcFile.resolvedFile != "" {
				return fmt.Errorf("cannot extend service %s of file %#v because this would cause an infinite loop. Please ensure your "+
					"docker compose services do not have a cyclical extends relationship", s.name, dcFile.resolvedFile)
			}
			return fmt.Errorf("cannot extend service %s (of the merged docker compose file) because this would cause an infinite loop. "+
				"Please ensure your docker compose services do not have a cyclical extends relationship", s.name)
		}
		return nil
	}
	s.visited = true
	if s.Extends == nil {
		return nil
	}
	s.recStack = true
	defer s.clearRecStack()
	sExtended, err := c.resolveExtends(s, dcFile)
	if err != nil {
		return err
	}
	merge(s, sExtended, false)
	return nil
}

// resolveExtends ensures the configuration of an extended docker compose service has been loaded.
// This may involve loading another file, and will recursively process any extends, erroring if a cycle is detected.
// More formally: given a docker compose service X named name, if X extends another service Y then resolveExtends ensures that:
// 1. the representation of Y has been loaded; -and
// 2. processExtends has been called on Y.
func (c *configLoader) resolveExtends(
	s *serviceInternal,
	dcFile *dockerComposeFile) (*serviceInternal, error) {
	var sExtended *serviceInternal
	var dcFileExtended *dockerComposeFile
	if s.Extends.File != nil {
		var err error
		dcFileExtended, err = c.loadFile(*s.Extends.File)
		if err != nil {
			return nil, err
		}
		// TODO https://github.com/kube-compose/kube-compose/issues/212 fail if there is a version mismatch
		sExtended = dcFileExtended.Services[s.Extends.Service]
		if sExtended == nil {
			return nil, extendsNotFoundError(s.name, dcFile.resolvedFile, s.Extends.Service, dcFileExtended.resolvedFile)
		}
	} else {
		dcFileExtended = dcFile
		sExtended = dcFile.Services[s.Extends.Service]
		if sExtended == nil {
			return nil, extendsNotFoundError(s.name, dcFile.resolvedFile, s.Extends.Service, dcFileExtended.resolvedFile)
		}
	}
	if sExtended.DependsOn != nil && len(sExtended.DependsOn.Values) > 0 {
		return nil, fmt.Errorf("cannot extend service %s: services with 'depends_on' cannot be extended",
			s.Extends.Service,
		)
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/122 perform full validation of extended service
	err := c.processExtends(sExtended, dcFileExtended)
	if err != nil {
		return nil, err
	}
	return sExtended, nil
}

func extendsNotFoundError(name1, file1, name2, file2 string) error {
	if file1 == "" {
		if file2 == "" {
			return fmt.Errorf(
				"a service named %s extends non-existent service %s (in merged docker compose files)",
				name1,
				name2,
			)
		}
		return fmt.Errorf(
			"a service named %s (in merged docker compose files) extends non-existent service %s (of file %#v)",
			name1,
			name2,
			file2,
		)
	}
	return fmt.Errorf(
		"a service named %s (of file %#v) extends non-existent service %s (of file %#v)",
		name1,
		file1,
		name2,
		file2,
	)
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
			dcFile, err := c.loadFile(file)
			if err != nil {
				return nil, err
			}
			resolvedFiles = append(resolvedFiles, dcFile.resolvedFile)
		}
	} else {
		var err error
		resolvedFiles, err = c.loadStandardFiles()
		if err != nil {
			return nil, err
		}
	}
	dcFileMerged, xProperties := c.merge(resolvedFiles)
	for _, s := range dcFileMerged.Services {
		err := c.processExtends(s, dcFileMerged)
		if err != nil {
			return nil, err
		}
	}
	err := resolveDependsOn(dcFileMerged.Services)
	if err != nil {
		return nil, err
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/165 resolve named volumes
	// TODO https://github.com/kube-compose/kube-compose/issues/166 error on duplicate mount points
	configCanonical := &CanonicalDockerComposeConfig{}
	configCanonical.Services = map[string]*Service{}
	for name, s := range dcFileMerged.Services {
		err = finalizeService(s)
		if err != nil {
			return nil, err
		}
		configCanonical.Services[name] = s.finalService
	}
	configCanonical.XProperties = xProperties
	return configCanonical, nil
}

func (c *configLoader) merge(resolvedFiles []string) (dcFileMerged *dockerComposeFile, xProperties []XProperties) {
	if len(resolvedFiles) > 1 {
		// TODO https://github.com/kube-compose/kube-compose/issues/213 error when trying to merge different versions
		// This if is not only an optimiziation (to avoid copying when there's only one service).
		// resolvedFile is "" in this case, which means that we can state "merged docker compose files" instead of a specific file in error
		// messages.
		dcFile := c.loadResolvedFileCache[resolvedFiles[0]].parsed
		dcFileMerged = &dockerComposeFile{
			Services: map[string]*serviceInternal{},
			version:  dcFile.version,
		}
		for i := len(resolvedFiles) - 1; i >= 0; i-- {
			dcFile := c.loadResolvedFileCache[resolvedFiles[i]].parsed
			mergeServices(dcFileMerged.Services, dcFile.Services)
			if dcFile.xProperties != nil {
				xProperties = append(xProperties, dcFile.xProperties)
			}
		}
	} else {
		dcFile := c.loadResolvedFileCache[resolvedFiles[0]].parsed
		dcFileMerged = dcFile
		if dcFile.xProperties != nil {
			xProperties = append(xProperties, dcFile.xProperties)
		}
	}
	return
}

func finalizeService(s *serviceInternal) error {
	if s.Command != nil {
		s.finalService.Command = s.Command.Values
	}
	if s.Entrypoint != nil {
		s.finalService.Entrypoint = s.Entrypoint.Values
	}
	s.finalService.Environment = s.environmentParsed

	// Healthchecks are processed after merging.
	healthcheck, healthcheckDisabled, err := ParseHealthcheck(s.Healthcheck)
	if err != nil {
		return err
	}
	s.finalService.Healthcheck = healthcheck
	s.finalService.HealthcheckDisabled = healthcheckDisabled

	if s.Image != nil {
		s.finalService.Image = *s.Image
	}
	s.finalService.Name = s.name
	s.finalService.Ports = s.portsParsed
	if s.Privileged != nil {
		s.finalService.Privileged = *s.Privileged
	}
	if s.Restart != nil {
		s.finalService.Restart = *s.Restart
	}
	s.finalService.User = s.User
	s.finalService.Volumes = s.Volumes
	if s.WorkingDir != nil {
		s.finalService.WorkingDir = *s.WorkingDir
	}
	return nil
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

func resolveDependsOn(services map[string]*serviceInternal) error {
	for _, s1 := range services {
		s1.finalService = &Service{}
	}
	for name1, s1 := range services {
		if s1.DependsOn != nil {
			for name2 := range s1.DependsOn.Values {
				s2 := services[name2]
				if s2 == nil {
					return fmt.Errorf("service %s refers to a non-existing service in its depends_on: %s", name1, name2)
				}
			}
			s1.finalService.DependsOn = s1.DependsOn.Values
		}
	}
	for _, s1 := range services {
		// Reset the visited marker on each service. This is a precondition of ensureNoDependsOnCycle.
		for _, s2 := range services {
			s2.visited = false
		}
		// Run the cycle detection algorithm...
		err := ensureNoDependsOnCycle(s1, services)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://www.geeksforgeeks.org/detect-cycle-in-a-graph/
func ensureNoDependsOnCycle(s1 *serviceInternal, services map[string]*serviceInternal) error {
	s1.visited = true
	s1.recStack = true
	defer s1.clearRecStack()
	if s1.DependsOn != nil {
		for name := range s1.DependsOn.Values {
			s2 := services[name]
			if !s2.visited {
				err := ensureNoDependsOnCycle(s2, services)
				if err != nil {
					return err
				}
			} else if s2.recStack {
				return fmt.Errorf("a service %s depends on a service %s, but this means there is a cycle in the depends_on relationship",
					s1.name, name)
			}
		}
	}
	return nil
}

// https://github.com/docker/compose/blob/master/compose/config/config_schema_v2.1.json
func (c *configLoader) parseDockerComposeFile(dcFile *dockerComposeFile) error {
	for name, s := range dcFile.Services {
		s.name = name
		err := c.parseDockerComposeFileService(dcFile, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *configLoader) parseDockerComposeFileService(dcFile *dockerComposeFile, s *serviceInternal) error {
	var err error
	s.portsParsed, err = parsePorts(s.Ports)
	if err != nil {
		return err
	}
	if s.Environment != nil {
		s.environmentParsed, err = c.parseEnvironment(s.Environment.Values)
		if err != nil {
			return err
		}
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/163 only resolve volume paths if volume_driver is not set.
	for i := 0; i < len(s.Volumes); i++ {
		resolveBindMountVolumeHostPath(dcFile.resolvedFile, &s.Volumes[i])
	}
	if s.Extends != nil && s.Extends.File != nil {
		*s.Extends.File = expandPath(dcFile.resolvedFile, *s.Extends.File)
	}
	return nil
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
