package config

import (
	"fmt"
	"reflect"
	"strings"

	version "github.com/hashicorp/go-version"
)

type ValueGetter func(name string) (string, bool)

type configInterpolator struct {
	config      genericMap
	errorList   []error
	valueGetter ValueGetter
	version     *version.Version
}

type stringOrInt struct {
	str   string
	i     int
	isInt bool
}

type path []stringOrInt

func (p path) appendStr(str string) path {
	return append(p, stringOrInt{
		str: str,
	})
}

func (p path) appendInt(i int) path {
	return append(p, stringOrInt{
		i:     i,
		isInt: true,
	})
}

func (p path) pop() path {
	return p[:len(p)-1]
}

func (c *configInterpolator) run() error {
	if !c.version.GreaterThan(v1) {
		c.interpolateSection(c.config, path{})
	} else {
		c.interpolateSectionByName("services")
		if !c.version.LessThan(v3_1) {
			c.interpolateSectionByName("volumes")
			c.interpolateSectionByName("networks")
			if !c.version.LessThan(v3_3) {
				c.interpolateSectionByName("secrets")
				c.interpolateSectionByName("configs")
			}
		}
	}
	// Primitive error handling does not report all errors...
	if len(c.errorList) > 0 {
		return c.errorList[0]
	}
	return nil
}

func (c *configInterpolator) interpolateSectionByName(name string) {
	if sectionRaw, ok := c.config[name]; ok {
		if section, ok := sectionRaw.(genericMap); ok {
			c.interpolateSection(section, (path{}).appendStr(name))
		}
	}
}

func (c *configInterpolator) interpolateSection(configDict genericMap, p path) {
	for keyRaw, val := range configDict {
		if key, ok := keyRaw.(string); ok {
			childPath := p.appendStr(key)
			val2 := c.interpolateRecursive(val, childPath)
			configDict[key] = val2
			childPath.pop()
		}
	}
}

func (c *configInterpolator) addError(err error, _ path) {
	c.errorList = append(c.errorList, err)
}

// InterpolateConfig takes the root of a docker compose file as a generic structure and substitutes variables in it.
// The implementation substitutes exactly the same sections as docker compose:
// https://github.com/docker/compose/master/compose/config/config.py.
// TODO https://github.com/jbrekelmans/kube-compose/issues/11 support arbitrary map types instead of genericMap.
func InterpolateConfig(config genericMap, valueGetter ValueGetter, v *version.Version) error {
	c := &configInterpolator{
		config:      config,
		valueGetter: valueGetter,
		version:     v,
	}
	return c.run()
}

type stringInterpolator struct {
	sb          strings.Builder
	str         string
	v           bool
	valueGetter ValueGetter
}

func (k *stringInterpolator) advance(n int) {
	k.str = k.str[n:]
}

func (k *stringInterpolator) processAfterDollarSignSimple() {
	// The grammar of names without braces is [a-zA-Z_][a-zA-Z0-9_]+
	// Scan until no letter, digit or underscore to find end of placeholder...
	i := 1
	for i < len(k.str) && (k.str[i] == '_' || IsASCIILetter(k.str[i]) || IsASCIIDigit(k.str[i])) {
		i++
	}
	value, found := k.valueGetter(k.str[0:i])
	if !found {
		value = ""
	}
	k.sb.WriteString(value)
	k.advance(i)
}

func (k *stringInterpolator) processCurlyBraceExpansionSimple(i int) {
	value, found := k.valueGetter(k.str[1:i])
	if !found {
		value = ""
	}
	k.sb.WriteString(value)
	k.advance(i + 1)
}

func (k *stringInterpolator) processCurlyBraceExpansionWithError(name, errorMsg string, treatEmptyAsUnset bool, i int) error {
	value, found := k.valueGetter(name)
	if !found || (value == "" && treatEmptyAsUnset) {
		return fmt.Errorf("substitution variable %#v has no value or value is empty: %#v", name, errorMsg)
	}
	k.sb.WriteString(value)
	k.advance(i + 1)
	return nil
}

func (k *stringInterpolator) processCurlyBraceExpansionWithDefault(name, defaultVal string, treatEmptyAsUnset bool, i int) {
	value, found := k.valueGetter(name)
	if !found || (value == "" && treatEmptyAsUnset) {
		value = defaultVal
	}
	k.sb.WriteString(value)
	k.advance(i + 1)
}

func (k *stringInterpolator) processCurlyBraceExpansion(i int) error {
	// Process what is between the two curly braces
	if k.v {
		j := strings.IndexAny(k.str[1:i], ":?-")
		if j >= 0 {
			j++
			switch {
			case k.str[j] == ':':
				switch {
				case k.str[j+1] == '?':
					return k.processCurlyBraceExpansionWithError(k.str[1:j], k.str[j+2:i], true, i)
				case k.str[j+1] == '-':
					k.processCurlyBraceExpansionWithDefault(k.str[1:j], k.str[j+2:i], true, i)
					return nil
				}
			case k.str[j] == '?':
				return k.processCurlyBraceExpansionWithError(k.str[1:j], k.str[j+1:i], false, i)
			default:
				k.processCurlyBraceExpansionWithDefault(k.str[1:j], k.str[j+1:i], false, i)
				return nil
			}
		}
	}
	k.processCurlyBraceExpansionSimple(i)
	return nil
}

func (k *stringInterpolator) processAfterDollarSign() error {
	if k.str[0] == '_' || IsASCIILetter(k.str[0]) {
		k.processAfterDollarSignSimple()
		return nil
	}
	if k.str[0] == '{' {
		// Scan until '}' to perform substitution...
		i := strings.IndexRune(k.str[1:], '}')
		if i < 0 {
			return fmt.Errorf("expected }")
		}
		i++
		return k.processCurlyBraceExpansion(i)
	}
	if k.str[0] == '$' {
		k.sb.WriteByte('$')
		k.advance(1)
		return nil
	}
	return fmt.Errorf("unexpected character after $")
}

func (k *stringInterpolator) getResult() string {
	if k.sb.Len() == 0 {
		// Fast path
		return k.str
	}
	// WriteString always returns a nil error
	k.sb.WriteString(k.str)
	return k.sb.String()
}

// Interpolate substitutes docker-compose style variables in the str.
// The docker-compose 2.1+ syntax is used if and only if version is true.
// The implementation is not strict on the syntax between two paired curly braces, but
// is otherwise identical to the Python implementation:
// https://github.com/docker/compose/blob/master/compose/config/interpolation.py
func Interpolate(str string, valueGetter ValueGetter, v bool) (string, error) {
	k := stringInterpolator{
		str:         str,
		v:           v,
		valueGetter: valueGetter,
	}
	for {
		i := strings.IndexRune(k.str, '$')
		if i < 0 {
			break
		}
		k.sb.WriteString(k.str[:i])
		k.advance(i + 1)
		if k.str == "" {
			return "", fmt.Errorf("$ followed by EOF")
		}
		err := k.processAfterDollarSign()
		if err != nil {
			return "", err
		}
	}
	return k.getResult(), nil
}

// IsASCIILetter returns true if and only if b is the ASCII code for a letter.
func IsASCIILetter(b byte) bool {
	return (byte('a') <= b && b <= byte('z')) || (byte('A') <= b && b <= byte('Z'))
}

// IsASCIIDigit returns true if and only if b is the ASCII code for a digit.
func IsASCIIDigit(b byte) bool {
	return byte('0') <= b && b <= byte('9')
}

func (c *configInterpolator) interpolateRecursive(obj interface{}, p path) interface{} {
	if str, ok := obj.(string); ok {
		str2, err := Interpolate(str, c.valueGetter, !c.version.LessThan(v2_1))
		if err != nil {
			c.addError(err, p)
		}
		return str2
	}
	if m, ok := obj.(genericMap); ok {
		for keyRaw, val := range m {
			if key, ok := keyRaw.(string); ok {
				childPath := p.appendStr(key)
				m[key] = c.interpolateRecursive(val, childPath)
				childPath.pop()
			}
		}
		return m
	}
	if obj != nil && reflect.TypeOf(obj).Kind() == reflect.Slice {
		slicev := reflect.ValueOf(obj)
		for i := 0; i < slicev.Len(); i++ {
			iv := slicev.Index(i)
			childPath := p.appendInt(i)
			val := iv.Interface()
			val2 := c.interpolateRecursive(val, childPath)
			iv.Set(reflect.ValueOf(val2))
			childPath.pop()
		}
	}
	return obj
}
